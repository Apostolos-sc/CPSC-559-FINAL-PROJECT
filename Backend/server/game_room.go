package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func handleGameConnection(db *sql.DB, conn net.Conn) {
	//each game Room Connection should be handled till the game is over
	//might wanna check and see if the game request is in proper format / incase of data corruption.
	//send message
	var timer = time.Now()
	var requestBuffer = make([]byte, 1024)
	//var responseBuffer = make([]byte, 1024)
	var accessCode string
	var err error
	var nRequest int
	for {
		//We can block here the request is handled with maybe a go routine. Let's think about it
		//Depends if we want to hold state on the server by closing connections and initiating connections
		//or just maintaining a pipeline of communication. (Go routines inside the go routine)
		nRequest, err = conn.Read(requestBuffer) // Blocking command
		if err != nil {
			log.Printf("There was an error while reading from the proxy the next command for the game Room.\n")
			//proxy replication here
		}
		//we should make sure the Protocol is enforced. There is some if statements missing here in each option
		var command = strings.Split(string(requestBuffer[:nRequest]), ":")
		var request = string(requestBuffer[:nRequest])
		var time_stamp int64
		log.Printf("Client Request: %s\n", request)
		//If message received is create room, send game Room Access Code
		if strings.Compare(command[0], "Create Room") == 0 {
			time_stamp, _ = strconv.ParseInt(command[2], 10, 64)
			//Create Room:username:TimeStamp is the communication format
			//Create a gameRoom in the database and then keep track in memory
			//Game Connection Attempt.
			//Call function that generates access code
			//add user to game room in database (command[1] stored in db)
			//Wait for proxy to send Acknowledgement
			player, queryErr := fetchRoomUser(db, command[1])
			if player != nil {
				log.Printf("Error while creating room. USER_IN_ROOM_ALREADY_ERROR.\n")
				_, err = conn.Write([]byte("USER_IN_ROOM_ALREADY_ERROR"))
				if err != nil {
					log.Printf("Failed to send USER_IN_ROOM_ALREADY_ERROR. %s\n", err.Error())
				}
			} else {
				if queryErr == sql.ErrNoRows {
					//username not found in the userRoom table, let's create a room.
					accessCode = generateAccessCode()
					//time.Sleep(8 * time.Second)
					//Room should be activated now
					gameRoomsMutex.Lock()
					//add game room to the Game Rooms map
					gameRooms[accessCode] = &gameRoom{accessCode: accessCode, currentRound: 0, numOfDisconnectedPlayers: 0, numOfPlayersAnsweredCorrect: 0, numOfPlayersAnswered: 0, accessCodeTimeStamp: -1, currentRoundTimeStamp: -1, numOfDisconnectedPlayersTimeStamp: -1, numOfPlayersAnsweredCorrectTimeStamp: -1, numOfPlayersAnsweredTimeStamp: -1, questions: make(map[int]*Question), players: make(map[string]*roomUser)}
					queryErr := insertGameRoom(db, gameRooms[accessCode])
					if queryErr != nil {
						if queryErr != nil {
							log.Printf("Error when inserting game Room information in the database, Command Executing : %s : %s\n", request, queryErr.Error())
						}
					}
					//Add player who created the room to the hash
					gameRooms[accessCode].players[command[1]] = &roomUser{username: command[1], accessCode: accessCode, points: 0, ready: 0, offline: 0, roundAnswer: 0, correctAnswer: -1, accessCodeTimeStamp: -1, pointsTimeStamp: -1, readyTimeStamp: -1, offlineTimeStamp: -1, roundAnswerTimeStamp: -1, correctAnswerTimeStamp: -1}
					queryErr = insertRoomUser(db, gameRooms[accessCode].players[command[1]])
					if queryErr != nil {
						log.Printf("Error when inserting Room User information in the database, Command Executing : %s : %s\n", request, queryErr.Error())
					} else {
						//No error
						log.Printf("Player with username %s has been added to the Game Room with access Code : %s\n", command[1], accessCode)
					}
					gameRoomsMutex.Unlock()
					//Send Response to Proxy with accessCode
					conn.Write([]byte("Room Created:" + accessCode))
					if err != nil {
						//Proxy replication
						log.Printf("Failed to send the game Room access Code to the proxy. %s\n", err.Error())
					} else {
						//successful sent to the proxy of access Code
						log.Printf("Game Room Access Code : %s was sent to the Proxy.\n", accessCode)
					}
					//Now we are ready to add access code to the valid game rooms
					log.Printf("Game room with Access Code : %s is active. Listening for requests....\n", accessCode)
				} else {
					//some other type of error.
				}
			}
		} else if strings.Compare(command[0], "Join Room") == 0 {
			time_stamp, _ = strconv.ParseInt(command[3], 10, 64)
			//Join Room:UserName:Access Code:TimeStamp
			//First check the game hashmap for the player
			gameRoomsMutex.Lock()
			//check if the player is part of the room already and also offline
			_, ok := gameRooms[command[2]].players[command[1]]
			if ok && gameRooms[command[2]].players[command[1]].offline == 1 {
				//don't have to query the db, if the player was added earlier to the room it will be in the hashmap of players.
				//player already in the room & is offline
				gameRooms[command[2]].players[command[1]].offline = 0
				//update database to reflect user reconnected
				updateRoomUser(db, gameRooms[command[2]].players[command[1]])
				elapsed := time.Since(timer).Seconds()
				var time_left string
				if float64(35)-elapsed < 0 {
					time_left = "0"
				} else {
					time_left = fmt.Sprintf("%f", float64(35)-elapsed)
				}
				gameRooms[command[2]].numOfDisconnectedPlayers--
				updateRoom(db, gameRooms[command[2]])
				//Remind danny to connect to time server upon reconnection
				conn.Write([]byte("Reconnect Success:{\"round\":\"" + strconv.Itoa(gameRooms[command[2]].currentRound) + "\",\"question\":\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].question + "\",\"answer\":\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].answer + "\", \"options\":[\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_1 + "\",\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_2 + "\", \"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_3 + "\", \"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_4 + "\"],\"time\":\"" + time_left + "\"}"))
				if err != nil {
					log.Printf("There was an error while sending Reconnect Succes to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
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
								gameRooms[command[2]].players[command[1]] = &roomUser{username: command[1], accessCode: accessCode, points: 0, ready: 0, offline: 0, roundAnswer: 0, correctAnswer: -1, accessCodeTimeStamp: -1, pointsTimeStamp: -1, readyTimeStamp: -1, offlineTimeStamp: -1, roundAnswerTimeStamp: -1, correctAnswerTimeStamp: -1}
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
								log.Printf("Game is underway, user %s cannot join room %s.\n", command[1], command[2])
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
			//format should be of Ready:Username:AccessCode:Response:TimeStamp
			time_stamp, _ = strconv.ParseInt(command[3], 10, 64)
			//update locally and in the database
			gameRoomsMutex.Lock()
			//since the map is loaded upon server promotion to Master, if the value is one, we don't need to update it
			//in the database, the server crashed before a response was received from the proxy
			if gameRooms[command[2]].players[command[1]].ready != 1 {
				gameRooms[command[2]].players[command[1]].ready = 1
				//Need testing, is there a chance we get 0 rows updated error in the new try if previously server crashed before proxy receives Ready Success response
				//should not get an error unless db is down
				queryErr := updateRoomUser(db, gameRooms[command[2]].players[command[1]])
				if queryErr != nil {
					log.Printf("There was an error while updating the user in the db.Error : %s", queryErr.Error())
					_, err = conn.Write([]byte("READY_USER_DB_UPDATE_ERROR"))
					if err != nil {
						//proxy couldn't receive error from server, proxy crashed probably
						log.Printf("There was an error while sending READY_USER_DB_UPDATE_ERROR to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
					} else {
						//error received from proxy
						log.Printf("Error READY_USER_DB_UPDATE_ERROR was sent to the proxy.\n")
					}
				}
			}
			//check if all users are ready
			var all_players_ready = true
			for key, element := range gameRooms[command[2]].players {
				if element.ready == 0 {
					log.Printf("User %s is not ready yet.\n", key)
					all_players_ready = false
					break
				}
			}
			if all_players_ready == true {
				//send to proxy all players ready
				if gameRooms[command[2]].currentRound != 1 {
					gameRooms[command[2]].currentRound = 1
					updateRoom(db, gameRooms[command[2]])
					//Check if game questions have already been generated -> this happens if server crashed previously right before sending
					//questions to proxy
				}
				//There is a very very rare chance, server crashed before all generated questions where inserted in db. Worry about later
				if len(gameRooms[command[2]].questions) < 10 {
					//Regenerate all questions - still need to check for rare case, will get insert errors probably, check later
					ok := generateQuestions(db, command[2])
					if ok {
						// Successfully generated questions and inserted them to the db.
						log.Printf("Successful generation of random questions for game room : %s.\n", accessCode)
					} else {
						// Error while generating a random set of questions for the game Room
						log.Printf("Failed to generate random questions for game room : %s.\n", accessCode)
					}
				}
				// If the server crashes here, we don't care, we just force a new timer in the time server, and
				// Replace with time server request, need to implement
				timer = time.Now()
				// No timestamps required as ready request can be rewritten
				_, err = conn.Write([]byte("All Ready:{\"round\":\"" + strconv.Itoa(gameRooms[command[2]].currentRound) + "\",\"question\":\"" + gameRooms[command[2]].questions[1].question + "\",\"answer\":\"" + gameRooms[command[2]].questions[1].answer + "\", \"options\":[\"" + gameRooms[command[2]].questions[1].option_1 + "\",\"" + gameRooms[command[2]].questions[1].option_2 + "\", \"" + gameRooms[command[2]].questions[1].option_3 + "\", \"" + gameRooms[command[2]].questions[1].option_4 + "\"]}"))
				if err != nil {
					//Proxy not connected, error, proxy replication
					log.Printf("There was an error while sending READY_USER_DB_UPDATE_ERROR to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
				} else {
					//successful sent of all questions to proxy
					log.Printf("Message All Ready:with all questions list was sent to the proxy.\n")
				}
			} else {
				//Send Ready success to proxy, There are more players that have not pressed ready
				_, err = conn.Write([]byte("Ready Success:" + command[1]))
				if err != nil {
					//Proxy disconnection detected, part of proxy replication - dw about it rn
					log.Printf("There was an error while sending Ready Success to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
				} else {
					//Successfully sent message to proxy
					log.Printf("Message Ready Success was sent to the proxy.\n")
				}
			}
			gameRoomsMutex.Unlock()
		} else if strings.Compare(command[0], "Answer") == 0 {
			time_stamp, _ = strconv.ParseInt(command[4], 10, 64)
			//format should be of Answer:Username:AccessCode:Response:TimeStamp
			var correct_answer = false
			gameRoomsMutex.Lock()
			log.Printf("Player answered : %s.\n", command[3])
			log.Printf("Corrent answer is : %s.\n", gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].answer)
			gameRooms[command[2]].players[command[1]].roundAnswer = gameRooms[command[2]].currentRound
			// Check if the user sends the correct answer
			if strings.Compare(command[3], gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].answer) == 0 {
				gameRooms[command[2]].players[command[1]].points += game_points[gameRooms[command[2]].numOfPlayersAnsweredCorrect]
				correct_answer = true
				gameRooms[command[2]].players[command[1]].correctAnswer = 1
				gameRooms[command[2]].numOfPlayersAnsweredCorrect++
			} else {
				gameRooms[command[2]].players[command[1]].correctAnswer = 0
			}
			updateRoomUser(db, gameRooms[command[2]].players[command[1]])
			gameRooms[command[2]].numOfPlayersAnswered++
			updateRoom(db, gameRooms[command[2]])
			log.Printf("Players Answered : %d.\n", gameRooms[command[2]].numOfPlayersAnswered)
			log.Printf("Total players in the game Room : %d.\n", len(gameRooms[command[2]].players))

			// Need to test if the player leaves after entering the answer, before at least one player is yet to answer - glitch
			if gameRooms[command[2]].numOfPlayersAnswered >= (len(gameRooms[command[2]].players) - gameRooms[command[2]].numOfDisconnectedPlayers) {
				if gameRooms[command[2]].currentRound < 10 {
					gameRooms[command[2]].currentRound++
					updateRoom(db, gameRooms[command[2]])
					var player_object_string = "\"leaderboard\":["
					for key, value := range gameRooms[command[2]].players {
						player_object_string = player_object_string + "{\"username\":\"" + key + "\",\"points\":\"" + strconv.Itoa(value.points) + "\"},"
					}
					_, err = conn.Write([]byte("Everyone Responded:{\"round\":\"" + strconv.Itoa(gameRooms[command[2]].currentRound) + "\",\"question\":\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].question + "\",\"answer\":\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].answer + "\", \"options\":[\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_1 + "\",\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_2 + "\", \"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_3 + "\", \"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_4 + "\"]," + player_object_string[:len(player_object_string)-1] + "]}"))
					if err != nil {
						log.Printf("There was an error while sending Everyone Responded to Proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
					} else {
						log.Printf("Message Everyone Responsed was sent to the proxy.\n")
					}
					timer = time.Now()
					gameRooms[command[2]].numOfPlayersAnswered = 0
					gameRooms[command[2]].numOfPlayersAnsweredCorrect = 0
					updateRoom(db, gameRooms[command[2]])
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
					break //Break out of the for loop
				}
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
			// Need to test for disconnection after the question is answered
		} else if strings.Compare(command[0], "Disconnect") == 0 {
			time_stamp, _ = strconv.ParseInt(command[3], 10, 64)
			//Disconnect:Username:AccessCode:TimeStamp
			//Proxy informs us that the client disconnected
			gameRoomsMutex.Lock()
			//let's check if game exists (was it deleted during a server crash?)
			_, ok := gameRooms[command[2]]
			if !ok {
				log.Printf("The room was previously deleted.")
				_, err = conn.Write([]byte("Successful Deletion of the Game Room."))
				if err != nil {
					log.Printf("There was an error while sending Successful Termination of game Room while underway to the proxy.")
					//proxy replication?
				} else {
					log.Printf("Successfully sent to proxy Successful Termination of Game Room.")
				}
			} else {
				//Check if the game is going or still in lobby
				if gameRooms[command[2]].currentRound > 0 && gameRooms[command[2]].currentRoundTimeStamp != time_stamp {
					if gameRooms[command[2]].numOfDisconnectedPlayersTimeStamp != time_stamp {

					}
					if len(gameRooms[command[2]].players)-gameRooms[command[2]].numOfDisconnectedPlayers-1 == 0 {
						//Last player in the room that disconnected. Let's terminate the game, delete users from room, room questions
						deleteRoomUsers(db, command[2])
						deleteRoomQuestions(db, command[2])
						deleteGameRoom(db, command[2])
						//delete room from hashmap
						delete(gameRooms, command[2])
						_, err = conn.Write([]byte("Successful Termination of Game Room while underway."))
						if err != nil {
							log.Printf("There was an error while sending Successful Termination of game Room while underway to the proxy.")
							//proxy replication?
						} else {
							log.Printf("Successfully sent to proxy Successful Termination of Game Room while underway.")
						}
						log.Printf("Game Room : %s has stopped servicing players.\n", command[2])
						conn.Close()
						gameRoomsMutex.Unlock()
						return
					} else {
						gameRooms[command[2]].players[command[1]].offline = 1
						updateRoomUser(db, gameRooms[command[2]].players[command[1]])
						gameRooms[command[2]].numOfDisconnectedPlayers++
						updateRoom(db, gameRooms[command[2]])
						var all_active_players_answered = true
						for _, value := range gameRooms[command[2]].players {
							if value.offline == 0 {
								if value.roundAnswer != gameRooms[command[2]].currentRound {
									all_active_players_answered = false
									break
								}
							}
						}
						if all_active_players_answered == true {
							//all active players answered, let's send next round information
							if gameRooms[command[2]].currentRound < 10 {
								gameRooms[command[2]].currentRound++
								updateRoom(db, gameRooms[command[2]])
								var player_object_string = "\"leaderboard\":["
								for key, value := range gameRooms[command[2]].players {
									player_object_string = player_object_string + "{\"username\":\"" + key + "\",\"points\":\"" + strconv.Itoa(value.points) + "\"},"
								}
								_, err = conn.Write([]byte("Everyone Responded:{\"round\":\"" + strconv.Itoa(gameRooms[command[2]].currentRound) + "\",\"question\":\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].question + "\",\"answer\":\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].answer + "\", \"options\":[\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_1 + "\",\"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_2 + "\", \"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_3 + "\", \"" + gameRooms[command[2]].questions[gameRooms[command[2]].currentRound].option_4 + "\"]," + player_object_string[:len(player_object_string)-1] + "]}"))
								if err != nil {
									log.Printf("There was an error while sending Everyone Responded to Proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
								} else {
									log.Printf("Message Everyone Responded was sent to the proxy.\n")
								}
								timer = time.Now()
								gameRooms[command[2]].numOfPlayersAnswered = 0
								gameRooms[command[2]].numOfPlayersAnsweredCorrect = 0
								updateRoom(db, gameRooms[command[2]])
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
								break //Break out of the for loop
							}
						} else {
							_, err = conn.Write([]byte("Game Disconnect:" + command[1]))
							if err != nil {
								log.Printf("There was an error while sending Lobby Disconnect:%s to proxy. Game Room: %s, Error: %s\n", command[1], command[2], err.Error())
							} else {
								log.Printf("Message LobbyDisconnect:%s was sent to the proxy.\n", command[1])
							}
						}
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
						_, err = conn.Write([]byte("Successful Termination of Game Room while in lobby."))
						if err != nil {
							log.Printf("There was an error while sending Successful Termination of game Room while in lobby to the proxy.")
							//proxy replication?
						} else {
							log.Printf("Successfully sent to proxy Successful Termination of Game Room while in lobby to the proxy.")
						}
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
							if value.offline != 1 {
								if value.ready != 1 {
									allPlayersReady = false
									log.Printf("Player %s is not ready to start after player disconnection.\n", key)
								} else {
									log.Printf("Player %s is ready to start after player disconnection.\n", key)
								}
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
							_, err = conn.Write([]byte("All Ready:{\"round\":\"" + strconv.Itoa(gameRooms[command[2]].currentRound) + "\",\"question\":\"" + gameRooms[command[2]].questions[1].question + "\",\"answer\":\"" + gameRooms[command[2]].questions[1].answer + "\", \"options\":[\"" + gameRooms[command[2]].questions[1].option_1 + "\",\"" + gameRooms[command[2]].questions[1].option_2 + "\", \"" + gameRooms[command[2]].questions[1].option_3 + "\", \"" + gameRooms[command[2]].questions[1].option_4 + "\"]}"))
							if err != nil {
								log.Printf("There was an error while sending All Ready message to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
							} else {
								log.Printf("Message All Ready was sent to the proxy.\n")
							}
							timer = time.Now()
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
			}
			gameRoomsMutex.Unlock()
		} else if strings.Compare(command[0], "Restart Room Connection") == 0 {
			//Command Format : Restart Room Connection:accessCode
			//lock the mutex
			accessCode = command[1]
			gameRoomsMutex.Lock()
			err = fetchRoom(db, command[1])
			if err != nil {
				log.Printf("There was an error while fetching the game Rooms. Error : %s.", err.Error())
			} else {
				err = fetchRoomUsers(db, command[1])
				if err != nil {
					log.Printf("There was an error while fetching the players participating in game room %s. Error : %s.", accessCode, err.Error())
				}
				err = fetchRoomQuestions(db, command[1])
				if err != nil {
					log.Printf("There was an error while fetching the game room Questions. Error : %s.", err.Error())
				}
				_, err = conn.Write([]byte("Successfully Loaded Previous Game & User Information."))
				if err != nil {
					log.Printf("Failed to send Successfully Loaded Previous Game & User Information. Error : %s", err.Error())
				}
			}

			// Unlock the mutex
			gameRoomsMutex.Unlock()
		} else {
			log.Println("Invalid Option Given by the proxy.")
		}
		log.Printf("Awaiting the next request for Game Room:%s ....\n", accessCode)
	}
}
