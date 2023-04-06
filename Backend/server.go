//Author         : Apostolos Scondrianis
//Created On     : 28-02-2023
//Last Edited By : Apostolos Scondrianis
//Last Edit On   : 02-03-2023
//Filename       : server.go
//Version        : 0.4

// Hashmap may be an issue? Maybe we want multiple servers to handle
// the same gameRoom incase one server is overly busy, think about
// during fault tolerance, replication stage, scalability
package main

import (
	"context"
	"database/sql"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Question struct {
	ID       int
	question string
	answer   string
	option_1 string
	option_2 string
	option_3 string
	option_4 string
}

// ready should be 0 or 1 to indicate if a player is ready to start
type roomUser struct {
	username   string
	accessCode string
	points     int
	ready      int
	offline    int
}
type connection struct {
	host     string
	port     string
	con_type string
}

type gameRoom struct {
	accessCode   string
	currentRound int
	questions    map[int]*Question
	players      map[string]*roomUser
}

var (
	gameRoomsMutex sync.Mutex
	gameRooms      map[string]*gameRoom
)

var MAX_PLAYERS int = 4
var MAX_ROUNDS int = 10
var PROXY = connection{"10.0.0.2", "9000", "tcp"}
var GAME_SERVICE = connection{"10.0.0.2", "8082", "tcp"}
var DB_master = connection{"10.0.0.2", "4406", "tcp"}
var DB_slave = connection{"10.0.0.2", "5506", "tcp"}
var db_master_user = "root"
var db_master_pw = "password"
var db_slave_user = "root"
var db_slave_pw = "password"
var game_points = [4]int{10, 9, 8, 7}

func main() {
	db, err := sql.Open("mysql", db_master_user+":"+db_master_pw+"@tcp("+DB_master.host+":"+DB_master.port+")/mydb")
	if err != nil {
		log.Printf("There was an DSN issue when opening the DB driver. Error : %s.\n", err.Error())
		//Handle error
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	defer db.Close()
	//Pinging db to check status
	if testConnection(db, DB_master.host, DB_master.port) {
		//successful test
	} else {
		//Handle error -> attempt to connect to slave
	}
	//handle db error

	db2, err := sql.Open("mysql", db_slave_user+":"+db_slave_pw+"@tcp("+DB_slave.host+":"+DB_slave.port+")/mydb")
	if err != nil {
		log.Printf("There was an DSN issue when opening the DB driver. Error : %s.\n", err.Error())
		//Handle error
	}
	db2.SetConnMaxLifetime(time.Minute * 3)
	db2.SetMaxOpenConns(10)
	db2.SetMaxIdleConns(10)

	defer db.Close()
	//Pinging db to check status

	if testConnection(db2, DB_slave.host, DB_slave.port) {
		//successful test
	} else {
		//Handle error -> attempt to connect to slave
	}
	defer db2.Close()

	if connectToProxy() {
		//Listen for Game Service Requests if you were successfully registered at the Proxy
		log.Println("Successful registration to proxy.")
		gameServiceTCPAddr, err := net.ResolveTCPAddr(GAME_SERVICE.con_type, GAME_SERVICE.host+":"+GAME_SERVICE.port)
		if err != nil {
			log.Printf("Unable to resolve Address for Listening for game connections : %s:%s error : %s\n", GAME_SERVICE.host, GAME_SERVICE.port, err.Error())
			//If we can't resolve address there is not much we can do on the server side. Might as well just shut er' down.s
			os.Exit(1)
		}

		// Start TCP Listener for the server
		listener, err := net.ListenTCP("tcp", gameServiceTCPAddr)
		if err != nil {
			log.Printf("Unable to start listener - at address : %s:%s, %s", GAME_SERVICE.host, GAME_SERVICE.port, err)
		} else {
			log.Printf("Listening on %v:%v\n", GAME_SERVICE.host, GAME_SERVICE.port)
		}
		//close Listener when the go routine is over
		//Will see if we can decide on a mechanism to start and shut down servers gracefully later.
		defer listener.Close()

		//Continuously Listen for game Connections
		for {
			//infinite listening - blocks while waiting in this go routine
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Printf("There was an error in Accepting the connection. Error : %s\n", err.Error())
				//Error with listener? Should we read from keyboard for IP Address and port to listen to ?
			}
			log.Printf("Incoming connection from : %s\n", conn.RemoteAddr().String())
			//Sub routine is called and we pass to it the connection parameter to be handled
			go handleGameConnection(db, conn)
		}
	} else {
		log.Println("Failed to register server at the proxy.")
	}
}

func handleGameConnection(db *sql.DB, conn net.Conn) {
	//each game Room Connection should be handled till the game is over
	//might wanna check and see if the game request is in proper format / incase of data corruption.
	//send message
	var playersAnswered int = 0
	var playersAnsweredCorrect int = 0
	var numOfDisconnectedPlayers int = 0
	var requestBuffer = make([]byte, 1024)
	var responseBuffer = make([]byte, 1024)
	var accessCode string
	var err error
	var nRequest int
	for {
		//We can block here the request is handled with maybe a go routine. Let's think about it
		//Depends if we want to hold state on the server by closing connections and initiating connections
		//or just maintaining a pipeline of communication. (Go routines inside the go routine)
		nRequest, err = conn.Read(requestBuffer)
		if err != nil {
			log.Printf("There was an error while reading from the proxy the next command for the game Room.\n")
			//proxy replication here
		}
		//we should make sure the Protocol is enforced. There is some if statements missing here in each option
		var command []string = strings.Split(string(requestBuffer[:nRequest]), ":")
		var request = string(requestBuffer[:nRequest])
		log.Printf("Client Request: %s\n", request)
		//If message received is create room, send game Room Access Code
		if strings.Compare(command[0], "Create Room") == 0 {
			//Create Room:username is the communication format
			//Create a gameRoom in the database and then keep track in memory
			//Game Connection Attempt.
			//Call function that generates access code
			//add user to game room in database (command[1] stored in db)
			//Wait for Acknowledgement
			player, queryErr := fetchRoomUser(db, command[1])
			if player != nil {
				log.Printf("Error while creating room. USER_IN_ROOM_ALREADY_ERROR.\n")
				conn.Write([]byte("USER_IN_ROOM_ALREADY_ERROR"))
				if err != nil {
					log.Printf("Failed to send USER_IN_ROOM_ALREADY_ERROR. %s\n", err.Error())
				}
			} else {
				if queryErr == sql.ErrNoRows {
					//username not found in the userRoom table, let's create a room.
					accessCode = generateAccessCode()
					//time.Sleep(8 * time.Second)
					conn.Write([]byte("Room Created:" + accessCode))
					if err != nil {
						log.Printf("Failed to send the game Room access Code to the proxy. %s\n", err.Error())
					}
					log.Printf("Game Room Access Code : %s was sent to the Proxy.\n", accessCode)
					//Wait for acknowledgement
					nResponse, err := conn.Read(responseBuffer)
					if err != nil {
						log.Printf("There was an error while receiving the Game Room Creation Acknowledgment from the proxy.\n")
					}
					if strings.Compare(string(responseBuffer[:nResponse]), "Access Code Received") == 0 {
						//Room should be activated now
						gameRoomsMutex.Lock()
						if gameRooms == nil {
							//map is unitiliazed
							gameRooms = make(map[string]*gameRoom)
						}
						//add game room to the Game Rooms map
						gameRooms[accessCode] = &gameRoom{accessCode: accessCode, currentRound: 0, questions: make(map[int]*Question), players: make(map[string]*roomUser)}
						queryErr := insertGameRoom(db, gameRooms[accessCode])
						if queryErr != nil {
							if queryErr != nil {
								log.Printf("Error when inserting game Room information in the database, Command Executing : %s : %s\n", request, queryErr.Error())
							}
						}
						//Add player to the hash
						gameRooms[accessCode].players[command[1]] = &roomUser{username: command[1], accessCode: accessCode, points: 0, ready: -1, offline: 0}
						queryErr = insertRoomUser(db, gameRooms[accessCode].players[command[1]])
						if queryErr != nil {
							if queryErr != nil {
								log.Printf("Error when inserting Room User information in the database, Command Executing : %s : %s\n", request, queryErr.Error())
							}
						}
						gameRoomsMutex.Unlock()
						log.Printf("Player with username %s has been added to the Game Room with access Code : %s\n", command[1], accessCode)
						log.Printf("Access Code : %s for the game Room was Received by the Proxy.\n", accessCode)
						//Now we are ready to add access code to the valid game rooms
						log.Printf("Game room with Access Code : %s is active. Listening for requests....\n", accessCode)
					} else {
						//Corrupt message from the proxy?? See if we need to handle this
						log.Printf("Proxy replied with %s\n", string(responseBuffer[:nResponse]))
					}
				} else {
					//some other type of error.
				}
			}
		} else if strings.Compare(command[0], "Join Room") == 0 {
			//Join Room:User Name:Access Code
			//First check the game hasmap for the player
			gameRoomsMutex.Lock()
			//check if the player is part of the room already and also offline
			_, ok := gameRooms[command[2]].players[command[1]]
			if ok && gameRooms[command[2]].players[command[1]].offline == 1 {
				//don't have to query the db, if the player was added earlier to the room it will be in the hashmap of players.
				//player already in the room & is offline
				gameRooms[command[2]].players[command[1]].offline = 0
				//update database to reflect user reconnected
				updateRoomUser(db, gameRooms[command[2]].players[command[1]])
				conn.Write([]byte("RECONNECT_SUCCESS"))
				if err != nil {
					log.Printf("There was an error while sending RECONNECT SUCCESS to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
				} else {
					log.Printf("Successfully reconnected user : %s to Game Room : %s\n", command[1], command[2])
				}
			} else {
				//query db to see if user is in another room
				player, queryErr := fetchRoomUser(db, command[1])
				if player != nil {
					log.Printf("Error while joining room. USER_IN_ANOTHER_ROOM_ERROR.\n")
					conn.Write([]byte("USER_IN_ANOTHER_ROOM_ERROR"))
					if err != nil {
						log.Printf("Failed to send USER_IN_ANOTHER_ROOM_ERROR. %s\n", err.Error())
					}
				} else {
					//let's check the type of queryError we got.
					if queryErr == sql.ErrNoRows {
						//user is not in any room
						if len(gameRooms[command[2]].players) < MAX_PLAYERS {
							if gameRooms[command[2]].currentRound == 0 {
								//player can join, there is room, assign it in memory
								gameRooms[command[2]].players[command[1]] = &roomUser{username: command[1], accessCode: accessCode, points: 0, ready: -1}
								//we need to also check for errors for the sql query
								insertRoomUser(db, gameRooms[command[2]].players[command[1]])
								log.Printf("Successfully added user : %s to Game Room : %s\n", command[1], command[2])
								_, err = conn.Write([]byte("JOIN_SUCCESS"))
								if err != nil {
									log.Printf("There was an error while sending JOIN SUCCESS to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
									//This will be handled by proxy replication when the time comes, we should assume proxy is down.
								} else {
									log.Printf("Successful sent JOIN_SUCCESS to user : %s\n", command[1])
								}
							} else {
								log.Printf("Game is underway, user %s cannot join room.\n", command[1], command[2])
								_, err = conn.Write([]byte("GAME_UNDERWAY"))
								if err != nil {
									log.Printf("There was an error while sending GAME_UNDERWAY to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
									//This will be handled by proxy replication when the time comes, we should assume proxy is down.
								} else {
									log.Printf("Successful sent GAME_UNDERWAY to proxy for %s's join request to Game Room %s.\n", command[1], command[2])
								}
							}
						} else {
							//The room is full send error to proxy
							_, err = conn.Write([]byte("ROOM_FULL"))
							if err != nil {
								log.Printf("There was an error while sending ROOM_FULL to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
							} else {
								log.Printf("Unable to add user to : %s to Game Room : %s. Room is full. ROOM_FULL sent to proxy.\n", command[1], command[2])
							}
						}
					} else {
						//potential DB problems
					}
				}
			}
			gameRoomsMutex.Unlock()
		} else if strings.Compare(command[0], "Ready") == 0 {
			//update locally and in the database
			gameRoomsMutex.Lock()
			//I need to convert my map of structs to maps of points to structs. Go lang maps change address all the time, so we are not
			//allowed to change the structs of maps directly.
			gameRooms[command[2]].players[command[1]].ready = 1
			queryErr := updateRoomUser(db, gameRooms[command[2]].players[command[1]])
			if queryErr != nil {
				log.Printf("There was an error while updating the user in the db.\n", queryErr.Error())
				_, err = conn.Write([]byte("READY_USER_DB_UPDATE_ERROR"))
				if err != nil {
					log.Printf("There was an error while sending READY_USER_DB_UPDATE_ERROR to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
				} else {
					log.Printf("Error READY_USER_DB_UPDATE_ERROR was sent to the proxy.\n")
				}
			} else {
				//check if all users are ready
				var all_players_ready bool = true
				for key, element := range gameRooms[command[2]].players {
					if element.ready == -1 {
						log.Printf("User %s is not ready yet.\n", key)
						all_players_ready = false
						break
					}
				}
				if all_players_ready == true {
					//send to proxy all players ready
					gameRooms[command[2]].currentRound = 1
					updateRoom(db, gameRooms[command[2]])
					ok := generateQuestions(db, command[2])
					if ok {
						log.Printf("Successful generation of random questions for game room : %s.\n", accessCode)
					} else {
						log.Printf("Failed to generate random questions for game room : %s.\n", accessCode)
					}
					_, err = conn.Write([]byte("All Ready:{\"question\":\"" + gameRooms[command[2]].questions[1].question + "\",\"answer\":\"" + gameRooms[command[2]].questions[1].answer + "\", \"options\":[\"" + gameRooms[command[2]].questions[1].option_1 + "\",\"" + gameRooms[command[2]].questions[1].option_2 + "\", \"" + gameRooms[command[2]].questions[1].option_3 + "\", \"" + gameRooms[command[2]].questions[1].option_4 + "\"]}"))
					if err != nil {
						log.Printf("There was an error while sending READY_USER_DB_UPDATE_ERROR to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
					} else {
						log.Printf("Message READY_ALL_PLAYERS_READY was sent to the proxy.\n")
					}
				} else {
					//send to proxy READY_SUCCESS
					_, err = conn.Write([]byte("Ready Success:" + command[1]))
					if err != nil {
						log.Printf("There was an error while sending Ready Success to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
					} else {
						log.Printf("Message Ready Success was sent to the proxy.\n")
					}
				}
			}
			gameRoomsMutex.Unlock()
		} else if strings.Compare(command[0], "Answer") == 0 {
			//format should be of Answer:Username:AccessCode:Response
			var correct_answer bool = false
			gameRoomsMutex.Lock()
			log.Printf("Player answered : %s.\n", command[3])
			log.Printf("Corrent answer is : %s.\n", gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].answer)
			if strings.Compare(command[3], gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].answer) == 0 {
				gameRooms[command[2]].players[command[1]].points += game_points[playersAnsweredCorrect]
				updateRoomUser(db, gameRooms[command[2]].players[command[1]])
				correct_answer = true
				playersAnsweredCorrect++
			}
			playersAnswered++
			log.Printf("Players Answered : %d.\n", playersAnswered)
			log.Printf("Total players in the game Room : %d.\n", len(gameRooms[command[2]].players))
			if playersAnswered == len(gameRooms[command[2]].players) {
				if gameRooms[command[2]].currentRound < 10 {
					gameRooms[command[2]].currentRound++
					updateRoom(db, gameRooms[command[2]])
					var player_object_string string = "\"leaderboard\":["
					for key, value := range gameRooms[command[2]].players {
						player_object_string = player_object_string + "{\"username\":\"" + key + "\",\"points\":\"" + strconv.Itoa(value.points) + "\"},"
					}
					_, err = conn.Write([]byte("Everyone Responded:{\"question\":\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].question + "\",\"answer\":\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].answer + "\", \"options\":[\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_1 + "\",\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_2 + "\", \"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_3 + "\", \"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_4 + "\"]," + player_object_string[:len(player_object_string)-1] + "]}"))
					if err != nil {
						log.Printf("There was an error while sending Everyone Responded to Proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
					} else {
						log.Printf("Message Everyone Responsed was sent to the proxy.\n")
					}
				} else {
					var player_object_string string = "{\"leaderboard\":["
					for key, value := range gameRooms[command[2]].players {
						player_object_string = player_object_string + "{\"username\":\"" + key + "\",\"points\":\"" + strconv.Itoa(value.points) + "\"},"
					}
					//We need to delete data from database
					for key, _ := range gameRooms[command[2]].players {
						deleteRoomUser(db, key)
					}
					deleteGameRoom(db, command[2])
					deleteRoomQuestions(db, command[2])
					_, err = conn.Write([]byte("Game Over:" + player_object_string[:len(player_object_string)-1] + "]" + "}"))
					if err != nil {
						log.Printf("There was an error while sending Game Over to Proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
						//proxy replication?
					} else {
						log.Printf("Message Game over:%s]} was sent to the proxy.\n", player_object_string[:len(player_object_string)-1])
					}
					log.Printf("Game Room %s communications will be terminated.\n", command[2])
					conn.Close()
					gameRoomsMutex.Unlock()
					break
				}
				playersAnswered = 0
				playersAnsweredCorrect = 0
			} else {
				if correct_answer {
					_, err = conn.Write([]byte("Correct Answer"))
					if err != nil {
						log.Printf("There was an error while sending Correct Answer to Proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
					} else {
						log.Printf("Message Correct Answer was sent to the proxy.\n")
					}
				} else {
					_, err = conn.Write([]byte("False Answer"))
					if err != nil {
						log.Printf("There was an error while sending False answer to the proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
					} else {
						log.Printf("Message False answer was sent to the proxy.\n")
					}
				}
			}
			gameRoomsMutex.Unlock()
		} else if strings.Compare(command[0], "Disconnect") == 0 {
			//Disconnect:Username:AccessCode
			//Proxy informs us that the client disconnected
			gameRoomsMutex.Lock()
			if gameRooms[command[2]].currentRound > 0 {
				if len(gameRooms[command[2]].players)-numOfDisconnectedPlayers-1 == 0 {
					//Last player in the room that disconnected. Let's terminate the game, delete users from room, room questions
					for key, _ := range gameRooms[command[2]].players {
						deleteRoomUser(db, key)
					}
					deleteRoomQuestions(db, command[2])
					deleteGameRoom(db, command[2])
					for key, _ := range gameRooms[command[2]].players {
						log.Printf("Deleting user %s from player hashmap for game %s", key, command[2])
						//delete player from hashmap
						delete(gameRooms[command[2]].players, key)
					}
					//delete room from hashmap
					delete(gameRooms, command[2])
					log.Printf("Game Room : %s has stopped servicing players.\n", command[2])
					conn.Close()
					gameRoomsMutex.Unlock()
					return
				} else {
					gameRooms[command[2]].players[command[1]].offline = 1
					updateRoomUser(db, gameRooms[command[2]].players[command[1]])
					numOfDisconnectedPlayers++
					//need to check these queries in case db flops
				}
			} else {
				log.Printf("Game has not started yet while user %s disconnected from room %s.\n", command[1], command[2])
				//Last player in the room that disconnected. Let's delete the room, delete users from room, and delete room questions
				if len(gameRooms[command[2]].players) == 1 {
					log.Printf("There is only one user in the room, let's delete him and his room.")
					//only one player in the room, delete the player
					deleteRoomUser(db, command[1])
					deleteRoomQuestions(db, command[2])
					deleteGameRoom(db, command[2])
					delete(gameRooms[command[2]].players, command[1])
					delete(gameRooms, command[2])
					gameRoomsMutex.Unlock()
					log.Printf("Game Room : %s has stopped servicing players.\n", command[2])
					conn.Close()
					return
				} else {
					log.Printf("There are more than one user in the room : %s\n", command[2])
					//delete the User
					deleteRoomUser(db, command[1])
					delete(gameRooms[command[2]].players, command[1])
					var allPlayersReady = true
					for key, value := range gameRooms[command[2]].players {
						if value.ready != 1 {
							allPlayersReady = false
							log.Printf("Player %s is not ready to start after player disconnection.\n", key)
						} else {
							log.Printf("Player %s is ready to start after player disconnection.\n", key)
						}
					}
					if allPlayersReady {
						//All players ready, start the game
						gameRooms[command[2]].currentRound = 1
						updateRoom(db, gameRooms[command[2]])
						ok := generateQuestions(db, command[2])
						if ok {
							log.Printf("Successful generation of random questions for game room : %s.\n", accessCode)
						} else {
							log.Printf("Failed to generate random questions for game room : %s.\n", accessCode)
						}
						_, err = conn.Write([]byte("All Ready:{\"question\":\"" + gameRooms[command[2]].questions[1].question + "\",\"answer\":\"" + gameRooms[command[2]].questions[1].answer + "\", \"options\":[\"" + gameRooms[command[2]].questions[1].option_1 + "\",\"" + gameRooms[command[2]].questions[1].option_2 + "\", \"" + gameRooms[command[2]].questions[1].option_3 + "\", \"" + gameRooms[command[2]].questions[1].option_4 + "\"]}"))
						if err != nil {
							log.Printf("There was an error while sending All Ready message to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
						} else {
							log.Printf("Message All Ready was sent to the proxy.\n")
						}
					} else {
						//not all players are ready, send disconnect message to proxy
						_, err = conn.Write([]byte("Lobby Disconnect:" + command[1]))
						if err != nil {
							log.Printf("There was an error while sending Lobby Disconnect:%s to proxy. Game Room: %s, Error: %s\n", command[1], command[2], err.Error())
						} else {
							log.Printf("Message LobbyDisconnect:%s was sent to the proxy.\n", command[1])
						}
					}
				}
			}
			gameRoomsMutex.Unlock()
		} else {
			log.Println("Invalid Option Given by the proxy.")
		}
		log.Printf("Awaiting the next request for Game Room:%s ....\n", accessCode)
	}
}

func generateAccessCode() string {
	//write code that generates access code, must be unique
	for {
		response, err := http.Get("https://www.random.org/integers?num=1&min=0&max=10000000&col=2&base=10&format=plain&rnd=new")

		if err != nil {
			log.Printf("There was an error while connecting to generate accessCode. Error: %s.\n", err.Error())
			os.Exit(1)
		}

		responseData, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Access Code was uniquely generated : %s", string(responseData))
		_, ok := gameRooms[strings.TrimSpace(string(responseData))]
		if ok {
			log.Printf("Room with access Code : %s exists. Will generate another one!.\n", strings.TrimSpace(string(responseData)))
		} else {
			log.Printf("Room with access Code : %s was generated!.\n", strings.TrimSpace(string(responseData)))
			return strings.TrimSpace(string(responseData))
		}

	}
}

func generateQuestions(db *sql.DB, accessCode string) bool {
	rand.Seed(time.Now().UnixNano())

	// Intn generates a random integer between 0 and 100
	// (not including 100)
	//read from DB the largest ID. and use that.
	var randomInts [10]int
	var maxQuestionsID int = 1428
	var questionsList [10]*Question
	for i := 0; i < 10; i++ {
		randomInts[i] = rand.Intn(maxQuestionsID)
		log.Printf("Random Questions ID %d for Room :%s generated.", randomInts[i], accessCode)
	}
	err := insertRoomQuestions(db, accessCode, randomInts[0], randomInts[1], randomInts[2], randomInts[3], randomInts[4], randomInts[5], randomInts[6], randomInts[7], randomInts[8], randomInts[9])
	if err != nil {
		//log the error return false
		log.Printf("There was an error when inserting the random generated Questions for room with accessCode : %s. Error : %s.\n", accessCode, err.Error())
		return false
	} else {
		for i := 0; i < 10; i++ {
			questionsList[i], err = fetchQuestion(db, randomInts[i])
			if err != nil {
				log.Printf("There was an issue when reading Question %d for game Room : %s. Error : %s.\n", i, accessCode, err.Error())
			} else {
				//assign from 1 to 10, 1 question for each round!
				gameRooms[accessCode].questions[i+1] = questionsList[i]
			}
		}
		//return true if we successfully added them to the database
		return true
	}
}

// Returns true if the proxy accepts the connection.
func connectToProxy() bool {
	//connection type, IpAddres:Port
	proxyAddr, err := net.ResolveTCPAddr(PROXY.con_type, PROXY.host+":"+PROXY.port)
	if err != nil {
		log.Println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}
	//attempt to connect to proxy using a tcp connection
	conn, err := net.DialTCP(PROXY.con_type, nil, proxyAddr)
	if err != nil {
		log.Println("Dial failed:", err.Error())
		os.Exit(1)
	}

	_, err = conn.Write([]byte("Server Join"))
	if err != nil {
		log.Println("Write data failed:", err.Error())
		os.Exit(1)
	}

	// buffer to get data
	received := make([]byte, 8192)
	n, err := conn.Read(received)
	if err != nil {
		log.Println("Read data failed:", err.Error())
		os.Exit(1)
	}
	if strings.Compare(string(received[:n]), "Accepted") == 0 {
		//If the proxy accepted us, send the address we will be serving at
		log.Printf("Received message: %s.\n", string(received[:n]))
		//Create a string IpAddress:PortNumber
		var gameServiceAddress = GAME_SERVICE.host + ":" + GAME_SERVICE.port
		_, err = conn.Write([]byte(gameServiceAddress))
		if err != nil {
			log.Println("Write data failed:", err.Error())
			os.Exit(1)
		} else {
			//wait for acknowledgement.
			n, err = conn.Read(received)
			if err != nil {
				log.Println("Read data failed:", err.Error())
				os.Exit(1)
			} else {
				//proxy received address success.
				if strings.Compare(string(received[:n]), "Received Address") == 0 {
					log.Printf("Received message: %s.\n", string(received[:n]))
					conn.Close()
					return true
				}
			}
		}
	}
	conn.Close()
	return false
}

func testConnection(db *sql.DB, host string, port string) bool {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	err := db.PingContext(ctx)
	if err != nil {
		log.Printf("Errors : %s, while pinging DB", err)
		return false
	} else {
		log.Printf("Successfully accessed database at %s:%s.\n", host, port)
		return true
	}
}

func updateRoom(db *sql.DB, room *gameRoom) error {
	query := "UPDATE gameRoom SET currentRound = ? WHERE accesscode = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, room.currentRound, room.accessCode)
	if err != nil {
		log.Printf("Error %s when updating gameRoom table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d rows updated gameRoom now contains information %s, %d", rows, room.accessCode, room.currentRound)
	return nil
}
func updateRoomUser(db *sql.DB, user *roomUser) error {
	query := "UPDATE roomUser SET points = ?, ready = ?, offline=? WHERE username = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, user.points, user.ready, user.offline, user.username)
	if err != nil {
		log.Printf("Error %s when updating roomUser table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d rows updated user now contains information %s, %s, %d, %d, %d.\n ", rows, user.username, user.accessCode, user.points, user.ready, user.offline)
	return nil
}

func insertRoomUser(db *sql.DB, player *roomUser) error {
	query := "INSERT INTO roomUser(username, accessCode, points, ready, offline) VALUES (?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, player.username, player.accessCode, player.points, player.ready, player.offline)
	if err != nil {
		log.Printf("Error %s when inserting row into roomUser table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d roomUser created with information %s, %s, %d, %d,  %d ", rows, player.username, player.accessCode, player.points, player.ready, player.offline)
	return nil
}

func insertGameRoom(db *sql.DB, room *gameRoom) error {
	query := "INSERT INTO gameRoom(accessCode, currentRound) VALUES (?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, room.accessCode, room.currentRound)
	if err != nil {
		log.Printf("Error %s when inserting row into gameRoom table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d gameRoom created with information %s, %d ", rows, room.accessCode, room.currentRound)
	return nil
}

func insertRoomQuestions(db *sql.DB, accessCode string, q1 int, q2 int, q3 int, q4 int, q5 int, q6 int, q7 int, q8 int, q9 int, q10 int) error {
	query := "INSERT INTO roomQuestions(accessCode, question_1_id,question_2_id,question_3_id,question_4_id,question_5_id,question_6_id,question_7_id,question_8_id,question_9_id,question_10_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, accessCode, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10)
	if err != nil {
		log.Printf("Error %s when inserting row into roomQuestions table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d roomQuestions created with information %s, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d.", rows, accessCode, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10)
	return nil
}

//Need to reimplement this!
/*
func fetchRoomQuestions(db *sql.DB, accessCode string) (error) {
    log.Printf("Getting roomQuestions for room %s.\n", accessCode)
    query := "SELECT * FROM roomQuestions where accessCode = ?"
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        log.Printf("Error %s when preparing SQL statement", err)
        return err
    }
    defer stmt.Close()
    var rQs roomQuestions
    row := stmt.QueryRowContext(ctx, accessCode)
    if err := row.Scan(&rQs.accessCode, &rQs.question_1, &rQs.question_2, &rQs.question_3, &rQs.question_4, &rQs.question_5, &rQs.question_6, &rQs.question_7, &rQs.question_8, &rQs.question_9, &rQs.question_10); err != nil {
        log.Printf("There was an error while fetching Room Questions for Room %s, Error : %s\n", accessCode, err.Error())
		return nil, err
    }
    return nil
}
*/
func fetchQuestion(db *sql.DB, ID int) (*Question, error) {
	log.Printf("Getting Question with ID : %d.\n", ID)
	query := "SELECT * FROM questions where question_id = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return nil, err
	}
	defer stmt.Close()
	var question Question
	row := stmt.QueryRowContext(ctx, ID)
	if err := row.Scan(&question.ID, &question.question, &question.answer, &question.option_1, &question.option_2, &question.option_3, &question.option_4); err != nil {
		log.Printf("There was an error while fetching Question with id %d, Error : %s\n", ID, err.Error())
		return nil, err
	} else {
		log.Printf("Successfully fetched question with information : %d, %s, %s, %s, %s, %s, %s.\n", question.ID, question.question, question.answer, question.option_1, question.option_2, question.option_3, question.option_4)
	}
	var question_mem *Question
	question_mem = new(Question)
	(*question_mem).ID = question.ID
	(*question_mem).question = question.question
	(*question_mem).answer = question.answer
	(*question_mem).option_1 = question.option_1
	(*question_mem).option_2 = question.option_2
	(*question_mem).option_3 = question.option_3
	(*question_mem).option_4 = question.option_4
	return question_mem, nil
}
func fetchRooms(db *sql.DB) ([]gameRoom, error) {
	log.Printf("Getting game")
	query := "select * from gameRoom;"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return []gameRoom{}, err
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return []gameRoom{}, err
	}
	defer rows.Close()
	var rooms = []gameRoom{}
	for rows.Next() {
		var room gameRoom
		if err := rows.Scan(&room.accessCode, &room.currentRound); err != nil {
			return []gameRoom{}, err
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return []gameRoom{}, err
	}
	return rooms, nil
}

func fetchRoomUsers(db *sql.DB, accessCode string) ([]roomUser, error) {
	log.Printf("Getting room Users.")
	query := "select * FROM roomUser WHERE accessCode = ?;"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return []roomUser{}, err
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx, accessCode)
	if err != nil {
		return []roomUser{}, err
	}
	defer rows.Close()
	var roomUsers = []roomUser{}
	for rows.Next() {
		var rUser roomUser
		if err := rows.Scan(&rUser.username, &rUser.accessCode, &rUser.points, &rUser.ready, &rUser.offline); err != nil {
			return []roomUser{}, err
		}
		roomUsers = append(roomUsers, rUser)
	}
	if err := rows.Err(); err != nil {
		return []roomUser{}, err
	}
	return roomUsers, nil
}

func fetchRoomUser(db *sql.DB, username string) (*roomUser, error) {
	log.Printf("Search for room user %s.\n", username)
	query := "SELECT * FROM roomUser where username = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return nil, err
	}
	defer stmt.Close()
	var player roomUser
	row := stmt.QueryRowContext(ctx, username)
	if err := row.Scan(&player.username, &player.accessCode, &player.points, &player.ready, &player.offline); err != nil {
		log.Printf("There was an error while fetching Room User %s, Error : %s\n", username, err.Error())
		return nil, err
	}
	var player_mem *roomUser
	player_mem = new(roomUser)
	(*player_mem).username = player.username
	(*player_mem).accessCode = player.accessCode
	(*player_mem).points = player.points
	(*player_mem).ready = player.ready
	(*player_mem).offline = player.offline
	return player_mem, nil
}

func deleteGameRoom(db *sql.DB, accessCode string) error {
	query := "DELETE FROM gameRoom WHERE accessCode = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err.Error())
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, accessCode)
	if err != nil {
		log.Printf("Error %s when deleting gameRoom %s.\n", err.Error(), accessCode)
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected.\n", err.Error())
		return err
	}
	log.Printf("Game Room %s was successfully deleted.\n", accessCode)
	return nil
}

func deleteRoomUser(db *sql.DB, username string) error {
	query := "DELETE FROM roomUser WHERE username = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err.Error())
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, username)
	if err != nil {
		log.Printf("Error %s when deleting row from roomUser table", err.Error())
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err.Error())
		return err
	}
	log.Printf("Room User %s was successfully deleted.\n", username)
	return nil
}

func deleteRoomQuestions(db *sql.DB, accessCode string) error {
	query := "DELETE FROM roomQuestions WHERE accessCode = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err.Error())
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, accessCode)
	if err != nil {
		log.Printf("Error %s when deleting gameRoom %s.\n", err.Error(), accessCode)
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected.\n", err.Error())
		return err
	}
	log.Printf("Game Room Questions %s were successfully deleted.\n", accessCode)
	return nil
}
