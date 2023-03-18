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
	"fmt"
	"log"
	"net"
	"os"
	"time"
	"strings"
	"sync"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
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

type roomQuestions struct {
	accessCode string
	question_1 int
	question_2 int
	question_3 int
	question_4 int
	question_5 int
	question_6 int
	question_7 int
	question_8 int
	question_9 int
	question_10 int
}

type roomUser struct  {
	username string
	accessCode string
	points int
}
type connection struct {
	host     string
	port     string
	con_type string
}

type gameRoom struct {
	accessCode string
	currentRound int
	players map[string]int
}

var (
	gameRoomsMutex sync.Mutex
	gameRooms      map[string]gameRoom
)

var MAX_PLAYERS int = 4
var PROXY = connection{"10.0.0.2", "9000", "tcp"}
var GAME_SERVICE = connection{"10.0.0.2", "8082", "tcp"}
var DB_master = connection{"10.0.0.2", "4406", "tcp"}
var DB_slave = connection{"10.0.0.2", "5506", "tcp"}
var db_master_user = "root"
var db_master_pw = "password"

func main() {
	db, err := sql.Open("mysql", db_master_user+":"+db_master_pw+"@tcp("+DB_master.host+":"+DB_master.port+")/mydb")
	if err != nil {
		fmt.Printf("There was an DSN issue when opening the DB driver. Error : %s.\n",  err.Error())
		//Handle error
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	
	defer db.Close()

    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()
    err = db.PingContext(ctx)
    if err != nil {
        log.Printf("Errors : %s, while pinging DB", err)
        return
    } else {
		fmt.Printf("Successfully connected to the database at %s:%s.\n", DB_master.host, DB_master.port)
	}
	results, err := db.Query("SELECT * FROM questions WHERE question_id >1419")
	if err != nil {
		fmt.Printf("There was an issue while performing query.\n", err.Error())
	}
	for results.Next() {
		var q Question
		// for each row, scan the result into our tag composite object
		err = results.Scan(&q.ID, &q.question, &q.answer, &q.option_1, &q.option_2, &q.option_3, &q.option_4)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}
		// and then print out the tag's Name attribute
		log.Printf(q.question)
	}
	if connectToProxy() {
		//Listen for Game Service Requests if you were successfully registered at the Proxy
		fmt.Println("Successful registration to proxy.")
		gameServiceTCPAddr, err := net.ResolveTCPAddr(GAME_SERVICE.con_type, GAME_SERVICE.host+":"+GAME_SERVICE.port)
		if err != nil {
			fmt.Printf("Unable to resolve Address for Listening for game connections : %s:%s error : %s\n", GAME_SERVICE.host, GAME_SERVICE.port, err.Error())
			//If we can't resolve address there is not much we can do on the server side. Might as well just shut er' down.s
			os.Exit(1)
		}

		// Start TCP Listener for the server
		listener, err := net.ListenTCP("tcp", gameServiceTCPAddr)
		if err != nil {
			fmt.Printf("Unable to start listener - at address : %s:%s, %s", GAME_SERVICE.host, GAME_SERVICE.port, err)
		} else {
			fmt.Printf("Listening on %v:%v\n", GAME_SERVICE.host, GAME_SERVICE.port)
		}
		//close Listener when the go routine is over
		//Will see if we can decide on a mechanism to start and shut down servers gracefully later.
		defer listener.Close()

		//Continuously Listen for game Connections
		for {
			//infinite listening - blocks while waiting in this go routine
			conn, err := listener.AcceptTCP()
			if err != nil {
				fmt.Printf("There was an error in Accepting the connection. Error : %s\n", err.Error())
				//Error with listener? Should we read from keyboard for IP Address and port to listen to ?
			}
			fmt.Printf("Incoming connection from : %s\n", conn.RemoteAddr().String())
			//Sub routine is called and we pass to it the connection parameter to be handled
			go handleGameConnection(conn)
		}
	} else {
		fmt.Println("Failed to register server at the proxy.")
	}
}

//Hashmaps have an access speed of O(1). So we decide to hold the accessCodes in hashmaps
//of the form key=> value : (accessCode => 1). If the accessCode is not found, assume that
//the server crashed and we are requesting from this server to assume responsibility of the
//game. Pull the game data from the database

//Note -> READY implementation potentially, or force start by host. Decide with frontend -> Danny
//Note -> At the end of each round when the clients send the answer, the server will send back the
//points and the next question.

//User attaches unique ID, after each command Request :userID at the end.

//What happens if the user disconnects. What do we do. username -> points

//Assumption for now that the user doesn't disconnect, he just keeps playing till game is over.
//generate UserID as unique identifier, screen name is chosen, can be the same with other people's name.
func handleGameConnection(conn net.Conn) {
	//each game Room Connection should be handled till the game is over
	//might wanna check and see if the game request is in proper format / incase of data corruption.
	//send message
	var accessCode string
	for {
		//We can block here the request is handled with maybe a go routine. Let's think about it
		//Depends if we want to hold state on the server by closing connections and initiating connections
		//or just maintaining a pipeline of communication. (Go routines inside the go routine)
		requestBuffer := make([]byte, 1024)
		responseBuffer := make([]byte, 1024)
		nRequest, err := conn.Read(requestBuffer)
		if err != nil {
			fmt.Printf("There was an error while reading from the proxy the next command for the game Room.\n")
		}
		//we should make sure the Protocol is enforced. There is some if statements missing here in each option
		var command []string = strings.Split(string(requestBuffer[:nRequest]), ":")
		var request = string(requestBuffer[:nRequest])
		fmt.Printf("Client Request: %s\n", request)
		//If message received is create room, send game Room Access Code
		if strings.Compare(command[0], "Create Room") == 0 {
			//Create Room:username is the communication format
			//Create a gameRoom in the database and then keep track in memory
			//Game Connection Attempt.
			//Call function that generates access code
			//add user to game room in database (command[1] stored in db)
			//Wait for Acknowledgement
			accessCode = generateAccessCode()
			//time.Sleep(8 * time.Second)
			conn.Write([]byte("Room Created:" + accessCode))
			if err != nil {
				fmt.Printf("Failed to send the game Room access Code to the proxy. %s\n", err.Error())
			}
			fmt.Printf("Game Room Access Code : %s was sent to the Proxy.\n", accessCode)
			//Wait for acknowledgement
			nResponse, err := conn.Read(responseBuffer)
			if err != nil {
				fmt.Printf("There was an error while receiving the Game Room Creation Acknowledgment from the proxy.\n")
			}
			if strings.Compare(string(responseBuffer[:nResponse]), "Access Code Received") == 0 {
				//Room should be activated now
				gameRoomsMutex.Lock()
				if gameRooms == nil {
					//map is unitiliazed
					gameRooms = make(map[string]gameRoom)
				}
				//add game room to the Game Rooms map
				gameRooms[accessCode] = gameRoom{players: make(map[string]int)}
				//Add player to the hash
				gameRooms[accessCode].players[command[1]] = 1
				gameRoomsMutex.Unlock()
				fmt.Printf("Player with username %s has been added to the Game Room with access Code : %s\n", command[1], accessCode)
				fmt.Printf("Access Code : %s for the game Room was Received by the Proxy.\n", accessCode)
				//Now we are ready to add access code to the valid game rooms
				fmt.Printf("Game room with Access Code : %s is active. Listening for requests....\n", accessCode)
			} else {
				//Corrupt message from the proxy?? See if we need to handle this
				fmt.Printf("Proxy replied with %s\n", string(responseBuffer[:nResponse]))
			}
		} else if strings.Compare(command[0], "Join Room") == 0 {
			//Join Room:User Name:Access Code
			//the command has 3 elements.
			//First check the game hasmap for the player
			gameRoomsMutex.Lock()
			//The way we have structured the game room it must be the case that it exists
			//Add player to the room
			//check if the player is part of the room already
			_, ok := gameRooms[command[2]].players[command[1]]
			if ok {
				//player already in the room, must be reconnection request
				conn.Write([]byte("RECONNECT_SUCCESS"))
				if err != nil {
					fmt.Printf("There was an error while sending RECONNECT SUCCESS to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
				} else {
					fmt.Printf("Successfully reconnected user : %s to Game Room : %s\n", command[1], command[2])
				}
			} else {
				//new player, check if room is full
				if len(gameRooms[command[2]].players) < MAX_PLAYERS {
					//player can join, there is room, assign it in memory
					gameRooms[command[2]].players[command[1]] = 1
					fmt.Printf("Successfully added user : %s to Game Room : %s\n", command[1], command[2])
					_, err = conn.Write([]byte("JOIN_SUCCESS"))
					if err != nil {
						fmt.Printf("There was an error while sending JOIN SUCCESS to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
						//This will be handled by proxy replication when the time comes, we should assume proxy is down.
					}
				} else {
					_, err = conn.Write([]byte("ROOM_FULL"))
					if err != nil {
						fmt.Printf("There was an error while sending ROOM FULL to proxy. Game Room: %s, Error: %s\n", command[2], err.Error())
					} else {
						fmt.Printf("Unable to add user to : %s to Game Room : %s. Room is full. ROOM FULL sent to proxy.\n", command[1], command[2])
					}
				}
			}
			gameRoomsMutex.Unlock()
		} else if strings.Compare(command[0], "Start Game") == 0 {
			//Requesting to start a game - has to be round 0
			if gameRooms == nil {
				//there are no rooms loaded check db
				//run db script to get gameRoom with accessCode selected
				gameRooms = make(map[string]gameRoom)
				//add accessCode once read
			} else {
				//we don't care about the value so _ representes that, ok will be true if the accessCode exists
				_, ok := gameRooms[command[1]]
				if ok {
					//return successfully joined
					//generate questions for the game Room
					if generateQuestions(command[1]) {
						//if successfully generated questions send
						//question to the proxy, and proxy sends questions to clients
					} else {
						//there was an issue with generating questions
						//send back ROOM_GENERATE_QUESTIONS_ERROR
					}
				} else {
					//accessCode doesn't exist in the map, access db and add to the hashmap if it's a real code
					//if it's not real there is an error. Return ROOM_CODE_ERROR
				}
			}

		} else if strings.Compare(command[0], "Stop Game") == 0 {
			//added arbirtrarily so the compiler doesn't complain. More logic to be done.
			//code is getting very long, hard to maintain. Might need to write functions for each
			//command
			// close conn
			conn.Close()
			break
		} else {
			fmt.Println("Invalid Option Given by the proxy.")
		}
		fmt.Printf("Awaiting the next request for Game Room:%s ....\n", accessCode)
	}
}

func generateAccessCode() string {
	//write code that generates access code, must be unique
	var accessCode string = "testCode"
	return accessCode
}

func generateQuestions(accessCode string) bool {
	//read DB and generate 10 random questions.
	//return true if we successfully added them to the database
	return true
}

//Returns true if the proxy accepts the connection.
func connectToProxy() bool {
	//connection type, IpAddres:Port
	proxyAddr, err := net.ResolveTCPAddr(PROXY.con_type, PROXY.host+":"+PROXY.port)
	if err != nil {
		fmt.Println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}
	//attempt to connect to proxy using a tcp connection
	conn, err := net.DialTCP(PROXY.con_type, nil, proxyAddr)
	if err != nil {
		fmt.Println("Dial failed:", err.Error())
		os.Exit(1)
	}

	_, err = conn.Write([]byte("Server Join"))
	if err != nil {
		fmt.Println("Write data failed:", err.Error())
		os.Exit(1)
	}

	// buffer to get data
	received := make([]byte, 8192)
	n, err := conn.Read(received)
	if err != nil {
		fmt.Println("Read data failed:", err.Error())
		os.Exit(1)
	}
	if strings.Compare(string(received[:n]), "Accepted") == 0 {
		//If the proxy accepted us, send the address we will be serving at
		fmt.Printf("Received message: %s.\n", string(received[:n]))
		//Create a string IpAddress:PortNumber
		var gameServiceAddress = GAME_SERVICE.host + ":" + GAME_SERVICE.port
		_, err = conn.Write([]byte(gameServiceAddress))
		if err != nil {
			fmt.Println("Write data failed:", err.Error())
			os.Exit(1)
		} else {
			//wait for acknowledgement.
			n, err = conn.Read(received)
			if err != nil {
				fmt.Println("Read data failed:", err.Error())
				os.Exit(1)
			} else {
				//proxy received address success.
				if strings.Compare(string(received[:n]), "Received Address") == 0 {
					fmt.Printf("Received message: %s.\n", string(received[:n]))
					conn.Close()
					return true
				}
			}
		}
	}
	conn.Close()
	return false
}

func insertRoomUser(db *sql.DB, username string, accessCode string, points int) error {
    query := "INSERT INTO roomUser(username, accessCode, points) VALUES (?, ?, ?)"
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        log.Printf("Error %s when preparing SQL statement", err)
        return err
    }
    defer stmt.Close()
    res, err := stmt.ExecContext(ctx, username, accessCode, points)
    if err != nil {
        log.Printf("Error %s when inserting row into roomUser table", err)
        return err
    }
    rows, err := res.RowsAffected()
    if err != nil {
        log.Printf("Error %s when finding rows affected", err)
        return err
    }
    log.Printf("%d roomUser created with information %s, %s, %d ", rows)
    return nil
}

func updateUserPoints(db *sql.DB, username string, points int) error {
    query := "UPDATE roomUser SET points = ? WHERE username = ?"
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        log.Printf("Error %s when preparing SQL statement", err)
        return err
    }
    defer stmt.Close()
    res, err := stmt.ExecContext(ctx, username, points)
    if err != nil {
        log.Printf("Error %s when inserting row into roomUser table", err)
        return err
    }
    rows, err := res.RowsAffected()
    if err != nil {
        log.Printf("Error %s when finding rows affected", err)
        return err
    }
    log.Printf("%d roomUser created with information %s, %s, %d ", rows)
    return nil
}

func insertGameRoom(db *sql.DB, accessCode string, currentRound int) error {
    query := "INSERT INTO gameRoom(accessCode, currentRound) VALUES (?, ?)"
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        log.Printf("Error %s when preparing SQL statement", err)
        return err
    }
    defer stmt.Close()
    res, err := stmt.ExecContext(ctx, accessCode, currentRound)
    if err != nil {
        log.Printf("Error %s when inserting row into gameRoom table", err)
        return err
    }
    rows, err := res.RowsAffected()
    if err != nil {
        log.Printf("Error %s when finding rows affected", err)
        return err
    }
    log.Printf("%d gameRoom created with information %s, %d ", rows, accessCode, currentRound)
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
    log.Printf("%d roomQuestions created with information %s, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d.", rows, accessCode, )
    return nil
}

func fetchRoomQuestions(db *sql.DB, accessCode string) (*roomQuestions, error) {
    fmt.Printf("Getting roomQuestions for room %s.\n", accessCode)
    query := "SELECT * FROM roomQuestions where accessCode = ?"
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        log.Printf("Error %s when preparing SQL statement", err)
        return nil, err
    }
    defer stmt.Close()
    var rQs roomQuestions
    row := stmt.QueryRowContext(ctx, accessCode)
    if err := row.Scan(&rQs.accessCode, &rQs.question_1, &rQs.question_2, &rQs.question_3, &rQs.question_4, &rQs.question_5, &rQs.question_6, &rQs.question_7, &rQs.question_8, &rQs.question_9, &rQs.question_10); err != nil {
        fmt.Printf("There was an error while fetching Room Questions for Room %s, Error : %s\n", accessCode, err.Error())
		return nil, err
    }
    return &rQs, nil
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
        if err := rows.Scan(&rUser.username, &rUser.accessCode, &rUser.points); err != nil {
            return []roomUser{}, err
        }
        roomUsers = append(roomUsers, rUser)
    }
    if err := rows.Err(); err != nil {
        return []roomUser{}, err
    }
    return roomUsers, nil
}