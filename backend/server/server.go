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
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

type connection struct {
	host     string
	port     string
	con_type string
}

type gameRoom struct {
	accessCode string
}

var (
	gameRoomsMutex sync.Mutex
	gameRooms      map[string]int
)

var MAX_PLAYERS int = 4
var PROXY = connection{os.Getenv("PROXY_HOST"), os.Getenv("PROXY_PORT"), "tcp"}
var GAME_SERVICE = connection{"127.0.0.1", "8080", "tcp"}

func main() {
	if connectToProxy() {
		//Listen for Game Service Requests if you were successfully registered at the Proxy
		fmt.Println("Successful registration to proxy.")
		gameServiceTCPAddr, err := net.ResolveTCPAddr(GAME_SERVICE.con_type, GAME_SERVICE.host+":"+GAME_SERVICE.port)
		if err != nil {
			fmt.Printf("Unable to resolve IP")
		}

		// Start TCP Listener for the server
		listener, err := net.ListenTCP("tcp", gameServiceTCPAddr)
		if err != nil {
			fmt.Printf("Unable to start listener - %s", err)
		} else {
			fmt.Printf("Listening on %v:%v\n", GAME_SERVICE.host, GAME_SERVICE.port)
		}
		//close Listener
		defer listener.Close()

		//Continuously Listen for game Connections
		for {
			//infinite listening
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
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

// Assumption for now that the user doesn't disconnect, he just keeps playing till game is over.
// generate UserID as unique identifier, screen name is chosen, can be the same with other people's name.
func handleGameConnection(conn net.Conn) {
	//each game Room Connection should be handled till the game is over
	//might wanna check and see if the game request is in proper format / incase of data corruption.
	//send message
	for {
		//We can block here the request is handled with maybe a go routine. Let's think about it
		//Depends if we want to hold state on the server by closing connections and initiating connections
		//or just maintaining a pipeline of communication. (Go routines inside the go routine)
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			log.Fatal(err)
		}
		//we should make sure the Protocol is enforced. There is some if statements missing here in each option
		var command []string = strings.Split(string(buffer[:n]), ":")
		//If message received is create room, send game Room Access Code
		if strings.Compare(command[0], "Create Room") == 0 {
			//Create Room:username is the communication format
			//Create a gameRoom in the database and then keep track in memory
			//Game Connection Attempt.
			//Call function that generates access code
			//add user to game room in database (command[1] stored in db)
			//Wait for Acknowledgement
			var accessCode string = generateAccessCode()
			conn.Write([]byte("Room Created:" + accessCode))
			if err != nil {
				fmt.Println("Write data failed:", err.Error())
				os.Exit(1)
			}
			fmt.Printf("Game Room Access Code : %s was sent to the Proxy.\n", accessCode)
			//Wait for acknowledgement
			n, err = conn.Read(buffer)
			if err != nil {
				log.Fatal(err)
			}
			if strings.Compare(string(buffer[:n]), "Access Code Received") == 0 {
				//Room should be activated now
				gameRoomsMutex.Lock()
				if gameRooms == nil {
					//map is unitiliazed
					gameRooms = make(map[string]int)
				}
				//add game room to the game rooms slice that the server tracks
				gameRooms[accessCode] = 1
				fmt.Printf("Access Code : %s for the game Room was Received by the Proxy.\n", accessCode)
				//Now we are ready to add access code to the valid game rooms
				fmt.Printf("Game room with Access Code : %s is active. Listening for requests....\n", accessCode)
			} else {
				//Corrupt message from the proxy??
				fmt.Printf("Proxy replied with %s\n", string(buffer[:n]))
			}
			//potential error handling here - think
		} else if strings.Compare(command[0], "Join Room") == 0 {
			//Check if a user wants to join a room. command[0] should contain the command and command[1] should contain the access code.
			//command[2] should contain the username
			if len(command) == 2 {
				//the command has 2 elements, let's check if the second element is a valid game room code.
				//First check the hasmap
				gameRoomsMutex.Lock()
				if gameRooms == nil {
					//there are no rooms loaded check db
					//run db script to get gameRoom with accessCode selected
					gameRooms = make(map[string]int)
					//add accessCode once read
				} else {
					//we don't care about the value so _ representes that, ok will be true if the accessCode exists
					_, ok := gameRooms[command[1]]
					if ok {
						//return successfully joined
						//add user to the room in the db, command[2] should be the username.
						//client side should make sure username is valid before sending
						//return JOIN_SUCCESS
					} else {
						//accessCode doesn't exist in the map, access db and add to the hashmap if it's a real code
						//if it's not real there is an error. Return ROOM_CODE_ERROR
					}
				}
				gameRoomsMutex.Unlock()
			} else if strings.Compare(command[0], "Start Game") == 0 {
				//Requesting to start a game - has to be round 0
				if gameRooms == nil {
					//there are no rooms loaded check db
					//run db script to get gameRoom with accessCode selected
					gameRooms = make(map[string]int)
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
				break
			}
			fmt.Println("Invalid Option Given by the proxy.")
		}
	}
	// close conn
	conn.Close()
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

// Returns true if the proxy accepts the connection.
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
