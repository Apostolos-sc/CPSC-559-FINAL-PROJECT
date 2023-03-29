//Author         : Apostolos Scondrianis
//Created On     : 28-02-2023
//Last Edited By : Apostolos Scondrianis
//Last Edit On   : 01-03-2023
//Filename       : proxy.go
//Version        : 0.2

package main

//Proxy
import (
	"reflect"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"github.com/gorilla/websocket"
)

//Connection information
type connection struct {
	host     string
	port     string
	con_type string
}

//gameRoom with an access Code and a server that will serve
type gameRoom struct {
	gameRoomConn *net.TCPConn
	players      map[string]*websocket.Conn
}

var client_counter = 0
//global ticker for tracking time intervals
//var ticker = time.NewTicker(2000 * time.Millisecond)

//Maximum number of game rooms a server should handle
var MAX_ROOMS_PER_SERVER int = 2

//Address to be listening for servers to indicate they want to serve
var SERVER_REGISTRATION = connection{"10.0.0.2", "9000", "tcp"}

//Address to be listening for clients
var CLIENT_SERVICE = connection{"10.0.0.2", "8000", "tcp"}

//Will be used to keep track of servers that are servicing
//make hashmap here when back from soccer
var (
	serverMutex       sync.Mutex
	serversSlice      []connection
	totalGamesServing []int
)

//Will be used to keep track of the gameRooms being serviced
var (
	gameRoomMutex sync.Mutex
	gameRooms     = make(map[string]gameRoom)
)

func main() {
	go serverListener()
	//go serverHealthChecks()
	clientListener()
}

// We'll need to define an Upgrader
// this will require a Read and Write buffer size
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}


func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	// upgrade this connection to a WebSocket Connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("There was an error when attempting to upgrade connection to a web socket. Error : %s\n", err.Error())
	}
	client_counter++
	handleClientRequest(ws, client_counter)
}

func clientListener() {
	//Client needs to provide which game room ID it is going be to connecting to.
	//websocket handler - no error handling needed here
	http.HandleFunc("/ws", wsEndpoint)
	//http listener
	log.Printf("Listening on %v:%v for Client Requests.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port)
	err := http.ListenAndServe(CLIENT_SERVICE.host+":"+CLIENT_SERVICE.port, nil)
	if err != nil {
		log.Printf("Unable to Listen and Serve HTTP Requests on %s:%s. Error : %s.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port, err.Error())
		//Need to handle error. Potential kill process.
	}
	log.Printf("Listening on %v:%v for client Requests.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port)

}

func serverListener() {
	// Resolve TCP Address
	//Address to be listening on
	serverRegistrationTCPAddr, err := net.ResolveTCPAddr(SERVER_REGISTRATION.con_type, SERVER_REGISTRATION.host+":"+SERVER_REGISTRATION.port)
	if err != nil {
		log.Printf("Unable to resolve IP address for server registration on the proxy server.\n")
	}

	// Start TCP Listener
	listener, err := net.ListenTCP("tcp", serverRegistrationTCPAddr)
	if err != nil {
		log.Printf("Unable to start the proxy listener listener - %s", err.Error())
	} else {
		log.Printf("Listening on %v:%v for Server Registration Requests.\n", SERVER_REGISTRATION.host, SERVER_REGISTRATION.port)
	}
	//close Listener
	defer listener.Close()

	//Continuously Listen for connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Server Listener Accept functionality error occurred:", err.Error())
			os.Exit(1)
		}
		log.Printf("Potential Server Registration Request Incoming from : %s\n", conn.RemoteAddr().String())
		go handleServerRegistration(conn)
	}
}

//handle Client Request. If Game Room pipeline is created maybe we need to set a timeout before
//a game room is utilized or just time out the room?
func handleClientRequest(clientConn *websocket.Conn, connID int) {
	var keepServicing bool = true
	var err error
	var n int
	var nResponse int
	buffer := make([]byte, 1024)
	serverResponseBuffer := make([]byte, 1024)
	for {
		if !keepServicing {
			break;
		}
		log.Printf("Waiting to read from client with ID %d.\n", connID)
		_, buffer, err = clientConn.ReadMessage()
		if err != nil {
			log.Println("Failed to read a request from the client. Connection will be terminated.")
			keepServicing = false
			continue
		} else {
			log.Printf("Message received from client with ID : %d.\n", connID)
		}
		n = len(buffer)
		//critical access
		serverMutex.Lock()
		if len(serversSlice) == 0 {
			serverMutex.Unlock()
			//Can't service Client, no live Servers.
			log.Println("There are no servers available to service Clients. Send Error to Client. Connection Terminated.")
			err = clientConn.WriteMessage(1, ([]byte("SERVER_AVAILABILITY_ERROR")))
			if err != nil {
				log.Println("Unable to send to client SERVER_AVAILABILITY_ERROR:", err.Error())
			}
			keepServicing = false
			continue
		}
		serverMutex.Unlock()
		var command []string = strings.Split(string(buffer[:n]), ":")
		var request = string(buffer[:n])
		//check if proxy received a valid request
		if checkRequest(command) {
			if strings.Compare(command[0], "Create Room") == 0 {
				done := false
				serverSelection := 0
				//do until the request is handled
				for !done {
					//arbitrary select the first server, will write algorithm later
					//to choose on with lightest number of game rooms handled
					//Client wants to create a game room - Handle game room
					//Critical Section
					gameRoomMutex.Lock()
					gameRoomAddr, err := net.ResolveTCPAddr("tcp", serversSlice[serverSelection].host+":"+serversSlice[serverSelection].port)
					address := string(serversSlice[serverSelection].host + ":" + serversSlice[serverSelection].port)
					gameRoomMutex.Unlock()
					if err != nil {
						log.Printf("ResolveTCPAddr failed for %s:%v\n", address, err.Error())
						//Need to handle the case where we can't resolve the server.
					}
					log.Printf("Create Room Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
					//attempt to connect to server to establish a game room connection using a tcp
					gameRoomConn, err := net.DialTCP("tcp", nil, gameRoomAddr)
					if err != nil {
						log.Printf("Dialing server %s for game Room Creation Request failed: %v\n", address, err.Error())
						//Need to handle the case where we can't resolve the server
					}
					//forward gameRoom creation request to server
					log.Printf("Requesting from Server with address %s Game Room Creation.\n", gameRoomConn.RemoteAddr().String())
					_, err = gameRoomConn.Write(buffer[:n])
					if err != nil {
						log.Printf("Sending the Game Room Creation Command : %s to the server failed: %s", string(buffer[:n]), err.Error())
						//Handle
					}
					//wait for response
					nResponse, err = gameRoomConn.Read(serverResponseBuffer)
					if err != nil {
						log.Println("Read data failed:", err.Error())
						log.Printf("Failed to receive a response from server : %s. Will attempt to connect to another server.\n", gameRoomConn.RemoteAddr().String())
						//failed to read from server, let's try to connect to another server and retry
						serverSelection++
						//assume server is down kill connection, need more logic to redirect all game Rooms serviced by that server
						//will fix soon
						gameRoomConn.Close()
						continue
					}
					/* Websocket messages
						// The message types are defined in RFC 6455, section 11.8.
						const (
							// TextMessage denotes a text data message. The text message payload is
							// interpreted as UTF-8 encoded text data.
							TextMessage = 1

							// BinaryMessage denotes a binary data message.
							BinaryMessage = 2

							// CloseMessage denotes a close control message. The optional message
							// payload contains a numeric code and text. Use the FormatCloseMessage
							// function to format a close message payload.
							CloseMessage = 8

							// PingMessage denotes a ping control message. The optional message payload
							// is UTF-8 encoded text.
							PingMessage = 9

							// PongMessage denotes a pong control message. The optional message payload
							// is UTF-8 encoded text.
							PongMessage = 10
						)
					*/
					var response []string = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
					if strings.Compare(response[0], "Room Created") == 0 {
						log.Printf("Game Room successfully created. Access Code : %s and is served by : %s\n", response[1], gameRoomConn.RemoteAddr().String())
						//add the game room to the gameRoom list tracked by the proxy
						//need to make sure there is no deadlock here - test with multiple game room creation requests at the same time.
						gameRoomMutex.Lock()
						serverMutex.Lock()
						//Initialize game Room Struct and initialize its map of players
						gameRooms[response[1]] = gameRoom{gameRoomConn, make(map[string]*websocket.Conn)}

						//assign the player with username command[1] to the gameRoom (He is the creator of the game Room)
						//so we should add him to the game Room since he was successful
						gameRooms[response[1]].players[command[1]] = clientConn
						totalGamesServing[0]++
						serverMutex.Unlock()
						gameRoomMutex.Unlock()
						//Send acknowledgement to server that proxy received access code
						_, err = gameRoomConn.Write([]byte("Access Code Received"))
						if err != nil {
							log.Printf("Sending Acknowledgement that the code was received to server %s failed : %s\n", gameRoomConn.RemoteAddr().String(), err.Error())
							//handle error
						}
						//Send to client success message.
						err = clientConn.WriteMessage(1, []byte("Access Code:"+response[1]))
						if err != nil {
							log.Printf("Write Access Code:%s to client %s failed: %s\n", response[1], clientConn.RemoteAddr().String(), err.Error())
							//Handle
						} else {
							log.Printf("Client with username %s & IP address %s received the Access Code.\n", command[1], clientConn.RemoteAddr().String())
						}
						done = true;
					} else if strings.Compare(response[0], "ROOM_CREATION_ERROR") == 0 {
						//Error with room creation -> potentially sent when db can't be reached? dunno we will see. Maybe not needed. This could violate
						//consistency if db can't be accessed and server crashes before changes are stored. So maybe server sends error if no DB can
						//be accessed
						err = clientConn.WriteMessage(1, []byte("Error:Room Creation"))
						if err != nil {
							log.Printf("Unable to send ROOM_CREATION_ERROR to client %s: %s\n", clientConn.RemoteAddr().String(), err.Error())
							//Handle
						} else {
							done = true
						}
					} else if strings.Compare(response[0], "USER_IN_ROOM_ALREADY_ERROR") == 0 {
						err = clientConn.WriteMessage(1, []byte("Error:User In Room Already"))
						if err != nil {
							log.Printf("Unable to send USER_IN_ROOM_ALREADY_ERROR to client %s: %s\n", clientConn.RemoteAddr().String(), err.Error())
							//Handle
						} else {
							done = true
						}
					} else {
						//Corrupt message? Or timeout handling here - figure it out later some recovery
						log.Printf("Server Response for Command : %s was %s.\n", string(buffer[:n]), string(serverResponseBuffer[:nResponse]))
					}
				}
			} else if strings.Compare(command[0], "Join Room") == 0 {
				//Client wants to Join a Room - First check if the
				//First let's check if the room exists - Lock Critical Resource first
				log.Printf("Join Room Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				gameRoomMutex.Lock()
				serverMutex.Lock()
				_, ok := gameRooms[command[2]]
				//Room exists
				if ok == true {
					_, ok_player := gameRooms[command[2]].players[command[1]]
					if(ok_player) {
						err = clientConn.WriteMessage(1, []byte("Error:Player is already active.")) 
						if err != nil {
							log.Printf("Unable to send Error:Player is already active.")
							//handle error
						}
					} else {
						//we need to contact the game server, send the command
						log.Printf("Requesting from Server with address %s a Join Room Request.\n", gameRooms[command[2]].gameRoomConn.RemoteAddr().String())
						_, err = gameRooms[command[2]].gameRoomConn.Write(buffer[:n])
						if err != nil {
							log.Printf("Sending Join Room Command %s to the server of game Room %s failed.\n", request, command[2])
							//Must Handle
						}
						nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
						if err != nil {
							log.Printf("Receiving Response from Server for Join Room Failed. SEND SERVER_RESPONSE_ERROR to client\n")
							err = clientConn.WriteMessage(1, []byte("Error:Server Response"))
							if err != nil {
								log.Printf("SERVER_RESPONSE_ERROR was not sent to the client. Error : \n", err.Error())
								//handle, client disconnected?
							}
						}
						join_response := string(serverResponseBuffer[:nResponse])
						if strings.Compare(join_response, "JOIN_SUCCESS") == 0 {
							var playerlist string = "Join Success:";
							gameRooms[command[2]].players[command[1]] = clientConn
							//Successful Join, let's send response to client
							log.Printf("Client %s with IP %s has successfully joined the Game Room %s.\n", command[1], clientConn.RemoteAddr().String(), command[2])
							for key := range gameRooms[command[2]].players { 
								if(strings.Compare(key, command[1]) != 0) {
									playerlist = playerlist+key+",";
								}
							}
							err = clientConn.WriteMessage(1, []byte(playerlist[:len(playerlist)-1]))
							if err != nil {
								log.Printf("There was an issue while sending Join Success acknowledgment to the client with IP address : %s. Error : %s \n", clientConn.RemoteAddr(), err.Error())
								//probably just close connection?
								//assume player disconnected?
							} else {
								//Send to the other clients that a user has joined!
								for key, value := range gameRooms[command[2]].players { 
									if(strings.Compare(key, command[1]) != 0) {
										value.WriteMessage(1, []byte("User Join:"+command[1]))
									}
								}
							}
						} else if strings.Compare(join_response, "RECONNECT_SUCCESS") == 0 {
							//need to do some testing. I think that if we kill the process the connection closes itself. Let's test
							//if not we need to close old connection
							//player, playerOk := gameRooms[command[2]].players[command[1]]
							//no need to contact game server, just reset the connection
							//Potential Error Point here -> if it doesn't work might need to replace with gameRooms[command[2]].players[command[1]]
							//assign the new connection as the player's connection
							gameRooms[command[2]].players[command[1]] = clientConn
							log.Printf("Player %s has successfully reconnected to game with access code %s.\n", command[1], command[2])
							var playerlist string = "";
							//send to the player who is currently in the lobby!
							for key := range gameRooms[command[2]].players { 
								if(strings.Compare(key, command[1]) != 0) {
									playerlist = playerlist+":"+key;
								}
							}
							err = clientConn.WriteMessage(1, []byte("Reconnect Success"+playerlist))
							if err != nil {
								log.Printf("There was an error with sending Reconnect Success to the client with IP : %s. Error : %s.\n", clientConn.RemoteAddr(), err.Error())
								//handle error
							} else {
								for key, value := range gameRooms[command[2]].players { 
									if(strings.Compare(key, command[1]) != 0) {
										value.WriteMessage(1, []byte("User Reconnect:"+command[1]))
									}
								}
							}
						} else if strings.Compare(join_response, "ROOM_FULL") == 0 {
							log.Printf("Player %s is unable to join Room %s. The room is currently full. Send to client Room Full.\n", command[1], command[2])
							err = clientConn.WriteMessage(1, []byte("Error:Room Full"))
							if err != nil {
								log.Printf("Room Full message was not sent to the client with IP address : %s. Error : %s \n", clientConn.RemoteAddr(), err.Error())
								//handle error do we assume player disconnected?
							}
						} else {
							log.Printf("Unexpected response from server for Join Room. Send to client with IP address : %s,  JOIN_ROOM_SERVER_ERROR\n", clientConn.RemoteAddr())
							err = clientConn.WriteMessage(1, []byte("Error:Unexpected Join Room Server Response"))
							if err != nil {
								log.Printf("JOIN_ROOM_SERVER_ERROR was not sent to the client. Error : \n", err.Error())
								//handle error do we assume player disconnected?
							}
						}
					}
				} else {
					//Room doesn't exist, send error, this will likely happen if data gets corrupt, highly unlikely
					log.Printf("Room with access Code %s doesn't exists. Join Non Existent Room Error was sent to the client.\n", command[2])
					log.Printf("%v\n", gameRooms);
					log.Printf("%s", reflect.TypeOf(command[2]))
					log.Printf("%v", ok);
					err = clientConn.WriteMessage(1, []byte("Error:Join Non Existent Room"))
					if err != nil {
						log.Printf("Join Non Existent Room Error was not sent to the client with IP addrees %s. Error : %s.\n", clientConn.RemoteAddr(), err.Error())
						//handle error do we assume player disconnected?
					}
				}
				serverMutex.Unlock()
				gameRoomMutex.Unlock()
			}else if strings.Compare(command[0], "Answer") == 0{
				//format should be of Answer:Username:AccessCode:Response Assume that the format of the request is correct.
				log.Printf("Answer Room Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				gameRoomMutex.Lock()
				serverMutex.Lock()
				_, err = gameRooms[command[2]].gameRoomConn.Write(buffer[:n]) 						
				if err != nil {
					log.Printf("Sending Answer Command %s to the server of game Room %s failed.\n", request, command[2])
				} else {
					nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
					if err != nil {
						log.Printf("Proxy was unable to receive a response from the server. Error : %s.\n", err.Error())
						//Replication
					} else {
						//Remember we need to start informing about disconnections
						answer_response := string(serverResponseBuffer[:nResponse])
						log.Printf("Server Responded : %s.\n", answer_response)
						var answer_command []string = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
						if strings.Compare(answer_command[0], "Everyone Responded") == 0 {
							for key, value := range gameRooms[command[2]].players { 
								err = value.WriteMessage(1, []byte(serverResponseBuffer[:nResponse])) 
								if err != nil {
									log.Printf("There was an issue while sending Every Player Responsed acknowledgment to the client with IP address : %s. Error : %s. \n", clientConn.RemoteAddr(), err.Error())
									//probably just close connection?
									//assume player disconnected?
									//need to inform server
								} else {
									log.Printf("Sent Everyone Responded to player %s.\n", key)
								}
							}
						}else if strings.Compare(answer_command[0], "Game Over") == 0 {
							for key, value := range gameRooms[command[2]].players { 
								err = value.WriteMessage(1, []byte(serverResponseBuffer[:nResponse])) 
								if err != nil {
									log.Printf("There was an issue while sending Game Over message to the client with IP address : %s. Error : %s. \n", clientConn.RemoteAddr(), err.Error())
									//probably just close connection?
									//assume player disconnected?
									//need to inform server
								} else {
									log.Printf("Sent Game Over to player %s.\n", key)
								}
							}
						} else {
							//no action here. we could use for leaderboard information during the game if needed
						}
					}
				}
				gameRoomMutex.Unlock()
				serverMutex.Unlock()
			} else if strings.Compare(string(buffer[:n]), "Start Game") == 0 {
				//Handle game start
			} else if strings.Compare(command[0], "Ready") == 0 {
				//Ready to start request from client
				log.Printf("Ready Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				if len(command) == 3 {
					//Command must be in the right format, let's work Ready:Username:AccessCode
					gameRoomMutex.Lock()
					serverMutex.Lock()
					_, ok := gameRooms[command[2]]
					//Room exists
					if ok == true {
						//we need to contact the game server, send the command
						log.Printf("Requesting from Server with address %s a Ready Request.\n", gameRooms[command[2]].gameRoomConn.RemoteAddr().String())
						_, err = gameRooms[command[2]].gameRoomConn.Write(buffer[:n])
						if err != nil {
							log.Printf("Sending Ready Command %s to the server of game Room %s failed.\n", request, command[2])
							//Must Handle
						}
						nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
						if err != nil {
							log.Printf("Receiving Response from Server for Ready Failed. Send Server Response Error to client.\n")
							err = clientConn.WriteMessage(1, []byte("Error:Server Failed to Respond"))
							if err != nil {
								log.Printf("Server Response Error was not sent to the client with IP Address %s. Error : %s.\n", clientConn.RemoteAddr(), err.Error())
								//handle, client disconnected?
							}
						} else {
							ready_response := string(serverResponseBuffer[:nResponse])
							var command []string = strings.Split(string(buffer[:n]), ":")
							var ready_command []string = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
							if strings.Compare(ready_command[0], "Ready Success") == 0 {
								//Successful READY, let's send response to client
								log.Printf("Client %s with IP %s has successfully set their status as ready to start the game.\n", command[1], clientConn.RemoteAddr().String())
								for key, value := range gameRooms[command[2]].players { 
										value.WriteMessage(1, []byte(string(serverResponseBuffer[:nResponse])))
										if err != nil {
											log.Printf("There was an issue while sending Ready Success to Player %s with IP address : %s. Error : %s. \n", key, clientConn.RemoteAddr(), err.Error())
											//probably just close connection?
											//assume player disconnected?
										}
								}
							} else if strings.Compare(ready_command[0], "All Ready") == 0 {
								//Successful READY, let's send response to client
								log.Printf("Client %s with IP %s has successfully set their status as ready to start the game and all players are ready to start.\n", command[1], clientConn.RemoteAddr().String())
								for key, value := range gameRooms[command[2]].players { 
									value.WriteMessage(1, []byte(string(serverResponseBuffer[:nResponse]))) 
									if err != nil {
										log.Printf("There was an issue while sending Ready Success All Players Ready Success acknowledgment to the client with IP address : %s. Error : %s. \n", clientConn.RemoteAddr(), err.Error())
										//probably just close connection?
										//assume player disconnected?
									}
									log.Printf("Sent Ready Success All Players Ready sent to player %s.\n", key)
								}
								//NEED ERROR HANDLING
							} else {
								log.Printf("The server sent the following response : %s\n", ready_response)
								err = clientConn.WriteMessage(1, []byte("Error:"+ready_response))
								if err != nil {
									log.Printf("There was an issue while sending the servers' answer to the client. \n")
									//probably just close connection?
									//assume player disconnected?
								}
							}
						}
					} else {
						//gameRoom doesn't exist
					}
					serverMutex.Unlock()
					gameRoomMutex.Unlock()
				} else {
					log.Printf("Ready Request %s has an incorrect format. Ready Request Format Error will be sent to the client with IP address : %s\n", clientConn.RemoteAddr().String())
					err = clientConn.WriteMessage(1, []byte("Error:Ready Request Invalid Format"))
					if err != nil {
						log.Println("Sending Ready Request Format to client with IP address : %s failed. Error : %s.", clientConn.RemoteAddr().String(), err.Error())
						//keepServicing = false
						//say goodbye maybe?
					}
				}
			} else {
				log.Printf("Request %s has an in correct format. COMMUNICATION_PROTOCOL_ERROR will be sent to the client.\n")
				err = clientConn.WriteMessage(1, []byte("Error:Invalid Command Sent"))
				if err != nil {
					log.Println("Sending COMMUNICATION_PROTOCOL_ERROR to client with IP address failed %s. Error : %s.", clientConn.RemoteAddr().String(), err.Error())
					keepServicing = false
				}
			}
		} else {
			//Cannot Service request, authentication information missing
			log.Printf("Communications Protocol Violated. Error will be sent to client, and connection terminated.\n")
			log.Printf("Command : %s\n", string(buffer[:n]))
			err = clientConn.WriteMessage(1, ([]byte("Error:Invalid Communication Protocol.")))
			if err != nil {
				log.Println("Sending COMMUNICATION_PROTOCOL_ERROR to client with IP address %s failed. Error %s.", clientConn.RemoteAddr().String(), err.Error())
				keepServicing = false
			}
		}
	}
	clientConn.Close()
}

func handleServerRegistration(conn net.Conn) {
	//Server Registration Handler
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("There was an issue with reading from the potential Server with IP %s. Error : %s.\n", conn.RemoteAddr(), err.Error())
	}
	if strings.Compare(string(buffer[:n]), "Server Join") == 0 {
		//Client attempting to connect is a server
		host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			log.Println("There was an error while splitting the remote Address of server. Error : %s.\n", err.Error())
		}
		//lock the servers_slice variable

		time := time.Now().Format(time.ANSIC)
		log.Printf("Command : %v. Send Accepted.\n", string(buffer[:]))
		conn.Write([]byte("Accepted"))
		n, err = conn.Read(buffer)
		if err != nil {
			log.Printf("There was an error while waiting to received the IP address on which the server with IP address : %s, will be servicing game rooms. Error : %s.\n", conn.RemoteAddr(), err.Error())
		}
		//We assume that the server next will send the IP address for which he will be listening on.
		host, port, err = net.SplitHostPort(string(buffer[:n]))
		if err != nil {
			log.Printf("There was an error while waiting extracting the IP address on which the server with IP address : %s, will be servicing game rooms. Error : %s.\n", conn.RemoteAddr(), err.Error())
		} else {
			log.Printf("Successfully extracted the address on which the server with IP address : %s, will be servicing game rooms on : %s:%s.\n", conn.RemoteAddr(), host,port)
			serverMutex.Lock()
			//Add server data on the server slice
			serversSlice = append(serversSlice, connection{host: host, port: port, con_type: "tcp"})
			//set total number of games serving to zero
			totalGamesServing = append(totalGamesServing, 0)
			serverMutex.Unlock()
			log.Printf("%s was added as a server on the server list on %v.\n", string(buffer[:n]), time)
			//send back that address was received to let know the server that all is OKAY
			_, err = conn.Write([]byte("Received Address"))
			if err != nil {
				log.Printf("There was an issue while sending Address Acknowledgement to the server with IP address : %s. Error : %s.\n", conn.RemoteAddr(), err.Error())
			}
		}
	} else {
		_, err = conn.Write([]byte("Wrong command given, access declined."))
		if err != nil {
			log.Printf("There was an issue while informing the potential server that the given command is incorrect. \n %s", err.Error())
		}
	}
	// close conn
	conn.Close()
}

func checkRequest(command []string) bool {
	if len(command) == 2 || len(command) == 3 || len(command) == 4 {
		if len(command) == 2 {
			if strings.Compare(command[0], "Create Room") == 0 {
				return true
			} else {
				return false
			}
		} else if len(command) == 3 {
			if strings.Compare(command[0], "Join Room") == 0 || strings.Compare(command[0], "Start Game") == 0 || strings.Compare(command[0], "Stop Game") == 0 || strings.Compare(command[0], "Ready") == 0{
				return true
			} else {
				return false
			}
		} else {
			if strings.Compare(command[0], "Answer") == 0 {
				return true;
			} else {
				return false;
			}
		}
	} else {
		return false
	}
}
