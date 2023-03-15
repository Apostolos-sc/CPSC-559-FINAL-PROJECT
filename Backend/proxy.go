//Author         : Apostolos Scondrianis
//Created On     : 28-02-2023
//Last Edited By : Apostolos Scondrianis
//Last Edit On   : 01-03-2023
//Filename       : proxy.go
//Version        : 0.2

package main

//Proxy
import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

//Connection information
type connection struct {
	host string;
	port string;
	con_type string;
}
//gameRoom with an access Code and a server that will serve
type gameRoom struct {
	gameRoomConn *net.TCPConn
	players map[string]*net.TCPConn
}

//global ticker for tracking time intervals
//var ticker = time.NewTicker(2000 * time.Millisecond)

//Maximum number of game rooms a server should handle
var MAX_ROOMS_PER_SERVER int = 2
//Address to be listening for servers to indicate they want to serve
var SERVER_REGISTRATION = connection {"10.0.0.2", "9000", "tcp"}
//Address to be listening for clients
var CLIENT_SERVICE = connection {"10.0.0.2", "8000", "tcp"}
//Will be used to keep track of servers that are servicing
//make hashmap here when back from soccer
var (
	serverMutex sync.Mutex
	serversSlice[] connection
	totalGamesServing[] int
)
//Will be used to keep track of the gameRooms being serviced
var (
	gameRoomMutex sync.Mutex
	gameRooms = make(map[string] gameRoom)
)

func main() {
	go serverListener()
	//go serverHealthChecks()
	clientListener()
}

/*func serverHealthChecks() {
	go func() {
		for range ticker.C {
			serverMutex.Lock()
			fmt.Println("Tick")
			severMutex.Unlock()
		}
	}()
}*/
func clientListener() {
	//Client needs to provide which game room ID it is going be to connecting to.
	clientServiceTCPAddr, err := net.ResolveTCPAddr(CLIENT_SERVICE.con_type, CLIENT_SERVICE.host+":"+CLIENT_SERVICE.port)
	if err != nil {
		fmt.Printf("Unable to resolve IP")
	}

	// Start TCP Listener
	listener, err := net.ListenTCP("tcp", clientServiceTCPAddr)
	if err != nil {
		fmt.Printf("Unable to start listener - %s", err)
	} else {
		fmt.Printf("Listening on %v:%v for client Requests.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port)
	}
	//close Listener
	defer listener.Close()

	//Continuously Listen for Client Connections
	for {
		//Serve Clients
		conn, err:= listener.AcceptTCP()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		fmt.Printf("Incoming Potential Client Service Request from : %s\n", conn.RemoteAddr().String())
		go handleClientRequest(conn)
	}
}

func serverListener () {
	// Resolve TCP Address
	//Address to be listening on
	serverRegistrationTCPAddr, err := net.ResolveTCPAddr(SERVER_REGISTRATION.con_type, SERVER_REGISTRATION.host+":"+SERVER_REGISTRATION.port)
	if err != nil {
		fmt.Printf("Unable to resolve IP")
	}

	// Start TCP Listener
	listener, err := net.ListenTCP("tcp", serverRegistrationTCPAddr)
	if err != nil {
		fmt.Printf("Unable to start listener - %s", err)
	} else {
		fmt.Printf("Listening on %v:%v for Server Registration Requests.\n", SERVER_REGISTRATION.host, SERVER_REGISTRATION.port)
	}
	//close Listener
	defer listener.Close()

	//Continuously Listen for connections
	for {
		conn, err:= listener.Accept()
		if err != nil {
			log.Fatal(err)
			fmt.Println("Server Listener Accept functionality error occurred:", err.Error())
			os.Exit(1)
		}
		fmt.Printf("Potential Server Registration Request Incoming from : %s\n", conn.RemoteAddr().String())
		go handleServerRegistration(conn)
	}
}
//handle Client Request. If Game Room pipeline is created maybe we need to set a timeout before
//a game room is utilized or just time out the room?
func handleClientRequest(clientConn *net.TCPConn) {
	var keepServicing bool = true
	for keepServicing {
		//Handle Client Request Here
		buffer := make([]byte, 1024)
		serverResponseBuffer := make([]byte, 1024)
		clientResponseBuffer := make([]byte, 1024)
		n, err := clientConn.Read(buffer)
		if err != nil {
			fmt.Println("Client has not sent an initial request. Connection will be terminated.")
			keepServicing = false
			continue
		}
		//critical access
		serverMutex.Lock()
		if(len(serversSlice) == 0) {
			serverMutex.Unlock()
			//Can't service Client, no live Servers.
			fmt.Println("There are no servers available to service Clients. Send Error to Client. Connection Terminated.")
			_, err = clientConn.Write(([]byte("SERVER_AVAILABILITY_ERROR")))
			if err != nil {
				fmt.Println("Unable to send to client SERVER_AVAILABILITY_ERROR:", err.Error())
			}
			keepServicing = false
			continue
		}
		serverMutex.Unlock()
		var command []string = strings.Split(string(buffer[:n]), ":")
		if(len(command) != 1 && len(command) >= 2) {
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
					address := string(serversSlice[serverSelection].host+":"+serversSlice[serverSelection].port)
					gameRoomMutex.Unlock()
					if err != nil {
						fmt.Printf("ResolveTCPAddr failed for %s:%v\n", address, err.Error())
						//Need to handle the case where we can't resolve the server.
					}
					fmt.Printf("User with username : %s & IP %s is asking for game Room Creation.\n", command[1], clientConn.RemoteAddr())
					//attempt to connect to server to establish a game room connection using a tcp
					gameRoomConn, err := net.DialTCP("tcp", nil, gameRoomAddr)
					if err != nil {
						fmt.Printf("Dialing server %s for game Room Creation Request failed: %v\n", address, err.Error())
						//Need to handle the case where we can't resolve the server
					}
					//forward gameRoom creation request to server
					fmt.Printf("Requesting from Server with address %s Game Room Creation.\n", gameRoomConn.RemoteAddr().String())
					DeadWriteErr := gameRoomConn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
					if DeadWriteErr != nil {
						fmt.Println("Unable to set Write Deadline Error.", DeadWriteErr)
					}
					_, err = gameRoomConn.Write([]byte(buffer[:n]))
					if err != nil {
						fmt.Printf("Sending the Game Room Creation Command : %s to the server failed: %s", string(buffer[:n]), err.Error())
						//Handle
					}
					//wait for response
					DeadReadErr := gameRoomConn.SetReadDeadline(time.Now().Add(5 * time.Second))
					if DeadReadErr != nil {
						fmt.Println("Unable to set read deadline for the server acknowledgment.")
					}
					nResponse, err := gameRoomConn.Read([]byte(serverResponseBuffer))
					if err != nil {
						fmt.Println("Read data failed:", err.Error())
						fmt.Printf("Failed to received a response from server : %s. Will attempt to connect to another server.\n", gameRoomConn.RemoteAddr().String())
						//failed to read from server, let's try to connect to another server and retry
						serverSelection++
						//assume server is down kill connection, need more logic to redirect all game Rooms serviced by that server
						//will fix soon
						gameRoomConn.Close()
						continue
					}
					var response []string = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
					if strings.Compare(response[0], "Room Created") == 0 {
						fmt.Printf("Game Room successfully created. Access Code : %s and is served by : %s", response[1], gameRoomConn.RemoteAddr().String())
						//add the game room to the gameRoom list tracked by the proxy
						//need to make sure there is no deadlock here - test with multiple game room creation requests at the same time.
						gameRoomMutex.Lock()
						serverMutex.Lock()
						//Initialize game Room Struct and initialize its map of players
						gameRooms[response[1]] = gameRoom{gameRoomConn, make(map[string]*net.TCPConn)}
						//assign the player with username command[1] to the gameRoom (He is the creator of the game Room)
						//so we should add him to the game Room since he was successful
						gameRooms[response[1]].players[command[1]] = clientConn
						keepErr := gameRoomConn.SetKeepAlive(true)
						if keepErr != nil {
							fmt.Printf("Unable to set keepalive - %s", err)	
						}
						gameRoomConn.SetKeepAlivePeriod(1 * time.Second)
						totalGamesServing[0]++
						serverMutex.Unlock()
						gameRoomMutex.Unlock()
						//Send acknowledgement to server that proxy received access code
						_, err = gameRoomConn.Write([]byte("Access Code Received"))
						if err != nil {
							fmt.Printf("Sending Acknowledgement that the code was received to server %s failed : %s\n", gameRoomConn.RemoteAddr().String(), err.Error())
							//handle error
						}
						//Send to client success message.
						_, err = clientConn.Write([]byte("Access Code:"+response[1]))
						if err != nil {
							fmt.Printf("Write Access Code:%s to client %s failed: %s\n", response[1], clientConn.RemoteAddr().String(), err.Error())
							//Handle
						}
						n, err = clientConn.Read([]byte(clientResponseBuffer))
						if err != nil {
							fmt.Printf("Reading Acknowledgement for Access Code Receive from Client %s: %s\n", clientConn.RemoteAddr().String(), err.Error())
							//Handle
						}
						if strings.Compare(string(clientResponseBuffer[:n]), "Access Code Received.") == 0 {
							//Acknowledgement received
							fmt.Printf("Client with username %s & IP address %s received the Access Code.\n", command[1], clientConn.RemoteAddr().String())
						} else{
							//neeeded? in case of corrupt message?
							fmt.Printf("Client send Access Code Acknowledgment Errorneous Message : %s", string(clientResponseBuffer[:n]))
							//Corrupt message or no acknowledgement? check. -> timeout implementation
						}
						done = true
					} else if strings.Compare(response[0], "ROOM_CREATION_ERROR") == 0 {
						//Error with room creation -> potentially sent when db can't be reached? dunno we will see. Maybe not needed. This could violate
						//consistency if db can't be accessed and server crashes before changes are stored. So maybe server sends error if no DB can
						//be accessed
						_, err = clientConn.Write([]byte("ROOM_CREATION_ERROR"))
						if err != nil {
							fmt.Printf("Unable to write to client %s: %s\n", clientConn.RemoteAddr().String(),err.Error())
							//Handle
						}
					} else {
						//Corrupt message? Or timeout handling here - figure it out later some recovery
						fmt.Printf("Server Response for Command : %s was %s.\n", string(buffer[:n]), string(serverResponseBuffer[:nResponse]))
					}
				}
			} else if strings.Compare(string(buffer[:n]), "Join Room") == 0 {
				//Client wants to Join a Room - First check if the
				if(len(command) != 3) {
					fmt.Printf("Join Room Command given %s by client %s, But must be formatted as : Join Room:Username:Access Code\n.", string(buffer[:n]), clientConn.RemoteAddr().String())
					_, err = clientConn.Write([]byte("JOIN_ROOM_ILLEGAL_PROTOCOL"))
					if err != nil {
						fmt.Printf("SEND JOIN_ROOM_ILLEGAL_PROTOCOL to client %s failed: %s\n", clientConn.RemoteAddr().String(), err.Error())
					}
					keepServicing = false;
					clientConn.Close()
				} else {
					//Request has three tokens, let's handle.
					//First let's check if the room exists - Lock Critical Resource first
					gameRoomMutex.Lock()
					serverMutex.Lock()
					_, ok := gameRooms[command[2]]
					//Room exists
					if ok == true {
						//check if the player exists already, i.e., are we reconnecting?
						player, playerOk := gameRooms[command[2]].players[command[1]]
						if playerOk == true {
							player.Close()
							//no need to contact game server, just reset the connection
							//Potential Error Point here -> if it doesn't work might need to replace with gameRooms[command[2]].players[command[1]]
							//assign the new connection as the player's connection
							player = clientConn
							fmt.Printf("Player %s has successfully reconnected to game with access code %s.\n", command[1], command[2])
							clientConn.Write([]byte("SUCCESSFUL_RECONNECTION"))
						} else {
							//we need to contact the game server, send the command
							_, err = gameRooms[command[2]].gameRoomConn.Write(buffer[:n])
							if err != nil {
								fmt.Printf("Sending Join Room Command %s to the server of game Room %s failed.\n", string(buffer[:n]), command[2])
							}
							nServerResponse, err := gameRooms[command[2]].gameRoomConn.Write([]byte(serverResponseBuffer))
							if err != nil {
								fmt.Printf("Receiving Response from Server for Join Room Failed. SEND SERVER_RESPONSE_ERROR\n",)
							} else {
								if strings.Compare(string(serverResponseBuffer[:nServerResponse]), "Join Success") == 0 {
									//Successful Join, let's send response to client
									_, err = clientConn.Write([]byte(serverResponseBuffer[:nServerResponse]))
									if err != nil {
										fmt.Printf("There was an issue while sending Join Success acknowledgment to the client. \n")
									}
								} else {
									//See what we can do later
									//There was an error with the acknowledgment potentially. Corrupt information or byzantine failure?
									fmt.Println("Join Success did not occur.")
								}
							}
						}
					} else {
						//Room doesn't exist, send error
						fmt.Printf("Room with access Code %s doesn't exists. JOIN_NON_EXISTENT_ROOM_ERROR was sent to the client.\n", command[2])
						clientConn.Write([]byte("JOIN_NON_EXISTENT_ROOM_ERROR"))
					}
					serverMutex.Unlock()
					gameRoomMutex.Unlock()
				}
			} else if strings.Compare(string(buffer[:n]), "Start Game") == 0 {
				//Handle game start
			}
		} else {
			//Cannot Service request, authentication information missing
			fmt.Printf("Communications Protocol Violated. Error will be sent to client, and connection terminated.\n")
			_, err = clientConn.Write(([]byte("COMMUNICATION_PROTOCOL_ERROR")))
			if err != nil {
				fmt.Println("Sending COMMUNICATION_PROTOCOL_ERROR to client failed:", err.Error())
				keepServicing = false;
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
		log.Fatal(err)
	}
	if strings.Compare(string(buffer[:n]), "Server Join") == 0 {
		//Client attempting to connect is a server
		host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			fmt.Println(err)
		}
		//lock the servers_slice variable

		time := time.Now().Format(time.ANSIC)
		fmt.Printf("Command : %v. Send Accepted.\n", string(buffer[:]))
		//responseStr := fmt.Sprintf("Accepted")
		conn.Write([]byte("Accepted"))
		n, err = conn.Read(buffer)
		if err != nil {
			log.Fatal(err)
		}
		//assume that the address that the server will be will listening for 
		//game room service requests
		host, port, err = net.SplitHostPort(string(buffer[:n]))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("")
			serverMutex.Lock()
			//Add server data on the server slice
			serversSlice = append(serversSlice, connection{host:host, port:port, con_type:"tcp"})
			//set total number of games serving to zero
			totalGamesServing = append(totalGamesServing, 0)
			serverMutex.Unlock()
			fmt.Printf("%s was added as a server on the server list on %v.\n", string(buffer[:n]), time)
			//send back that address was received to let know the server that all is OKAY
			conn.Write([]byte("Received Address"))
		}
	} else {
		conn.Write([]byte("Wrong command given, access declined."))
	}
	// close conn
	conn.Close()
}