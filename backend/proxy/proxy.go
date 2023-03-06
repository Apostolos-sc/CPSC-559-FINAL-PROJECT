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

// Connection information
type connection struct {
	host     string
	port     string
	con_type string
}

// gameRoom with an access Code and a server that will serve
type gameRoom struct {
	accessCode string
	conn       *net.TCPConn
}

// Maximum number of game rooms a server should handle
var MAX_ROOMS_PER_SERVER int = 2

// Address to be listening for servers to indicate they want to serve
var SERVER_REGISTRATION = connection{os.Getenv("HOST"), os.Getenv("PORT"), "tcp"}

// Address to be listening for clients
var CLIENT_SERVICE = connection{os.Getenv("HOST"), "8000", "tcp"}

// Will be used to keep track of servers that are servicing
// make hashmap here when back from soccer
var (
	serverMutex       sync.Mutex
	serversSlice      []connection
	totalGamesServing []int
)

// Will be used to keep track of the gameRooms being serviced
var (
	gameRoomMutex  sync.Mutex
	gameRoomsSlice []gameRoom
)

func main() {
	go serverListener()
	clientListener()
}

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
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		fmt.Printf("Incoming Potential Client Service Request from : %s\n", conn.RemoteAddr().String())
		go handleClientRequest(conn)
	}
}

func serverListener() {
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
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		fmt.Printf("Potential Server Registration Request Incoming from : %s\n", conn.RemoteAddr().String())
		go handleServerRegistration(conn)
	}
}

// handle Client Request. If Game Room pipeline is created maybe we need to set a timeout before
// a game room is utilized or just time out the room?
func handleClientRequest(clientConn net.Conn) {
	//Handle Client Request Here
	buffer := make([]byte, 1024)
	n, err := clientConn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	//critical access
	serverMutex.Lock()
	if len(serversSlice) == 0 {
		serverMutex.Unlock()
		//Can't service Client, no live Servers.
		fmt.Println("There are no servers available to service Clients. Send Error to Client. Connection Terminated.")
		_, err = clientConn.Write(([]byte("SERVER_AVAILABILITY_ERROR")))
		if err != nil {
			fmt.Println("Write data failed:", err.Error())
			os.Exit(1)
		}
		clientConn.Close()
		return
	}
	serverMutex.Unlock()
	var command []string = strings.Split(string(buffer[:n]), ":")
	if len(command) != 1 {
		if strings.Compare(command[0], "Create Room") == 0 {
			//Check if username was sent with the request before creating room (NOT DONE!)
			//arbitrary select the first server, will write algorithm later
			//to choose on with lightest number of game rooms handled
			//Client wants to create a game room - Handle game room
			//Critical Section
			gameRoomMutex.Lock()
			gameRoomAddr, err := net.ResolveTCPAddr("tcp", serversSlice[0].host+":"+serversSlice[0].port)
			gameRoomMutex.Unlock()
			if err != nil {
				fmt.Println("ResolveTCPAddr failed:", err.Error())
				os.Exit(1)
			}
			fmt.Printf("User with username : %s & IP %s is asking for game Room Creation.\n", command[1], clientConn.RemoteAddr())
			//attempt to connect to server to establish a game room connection using a tcp
			gameRoomConn, err := net.DialTCP("tcp", nil, gameRoomAddr)
			if err != nil {
				fmt.Println("Dial failed:", err.Error())
				os.Exit(1)
			}
			//forward gameRoom creation request to server
			fmt.Printf("Requesting from Server with address %s Game Room Creation.\n", gameRoomConn.RemoteAddr().String())
			_, err = gameRoomConn.Write([]byte(buffer[:n]))
			if err != nil {
				fmt.Println("Write data failed:", err.Error())
				os.Exit(1)
			}
			//wait for response
			n, err = gameRoomConn.Read([]byte(buffer))
			if err != nil {
				fmt.Println("Read data failed:", err.Error())
				os.Exit(1)
			}
			var response []string = strings.Split(string(buffer[:n]), ":")
			if strings.Compare(response[0], "Room Created") == 0 {
				fmt.Printf("Game Room successfully created. Access Code : %s and is served by : %s", response[1], gameRoomConn.RemoteAddr().String())
				//add the game room to the gameRoom list tracked by the proxy
				//need to make sure there is no deadlock here - test with multiple game room creation requests at the same time.
				gameRoomMutex.Lock()
				serverMutex.Lock()
				//store information about the game Room session in the proxy's memory
				gameRoomsSlice = append(gameRoomsSlice, gameRoom{response[1], gameRoomConn})
				totalGamesServing[0]++
				serverMutex.Unlock()
				gameRoomMutex.Unlock()
				//Send acknowledgement to server that proxy received access code
				_, err = gameRoomConn.Write([]byte("Access Code Received"))
				if err != nil {
					fmt.Println("Write data failed:", err.Error())
					os.Exit(1)
				}
				//Send to client success message.
				_, err = clientConn.Write([]byte("Access Code:" + response[1]))
				if err != nil {
					fmt.Println("Write data failed:", err.Error())
					os.Exit(1)
				}
				n, err = clientConn.Read([]byte(buffer))
				if err != nil {
					fmt.Println("Read data failed:", err.Error())
					os.Exit(1)
				}
				if strings.Compare(string(buffer[:n]), "Access Code Received.") == 0 {
					//Acknowledgement received
					fmt.Printf("Client with username %s & IP address %s received the Access Code.\n", command[1], clientConn.RemoteAddr().String())
				} else {
					//neeeded? -> clean up
					fmt.Printf("Something went wrong. Deal with it Programmer :P\n")
					//Corrupt message or no acknowledgement? check. -> timeout implementation
				}
			} else if strings.Compare(response[0], "ROOM_CREATION_ERROR") == 0 {
				//Error with room creation -> potentially sent when db can't be reached? dunno we will see. Maybe not needed. This could violate
				//consistency if db can't be accessed and server crashes before changes are stored. So maybe server sends error if no DB can
				//be accessed
				_, err = clientConn.Write([]byte("Unable to create the Game Room. An error has occurred."))
				if err != nil {
					fmt.Println("Write data failed:", err.Error())
					os.Exit(1)
				}
			} else {
				//Corrupt message? Or timeout handling here - figure it out later some recovery
				fmt.Printf("Server Message corrup.\n")
			}
		} else if strings.Compare(string(buffer[:n]), "Join Room") == 0 {
			//Client wants to Join a Room
		} else if strings.Compare(string(buffer[:n]), "Start Game") == 0 {
			//Handle game start
		}
		//done handling client, bye bye
	} else {
		//Cannot Service request, authentication information missing
		fmt.Printf("Communications Protocol Violated. Error will be sent to client, and connection terminated.\n")
		_, err = clientConn.Write(([]byte("REQUEST_PROTOCOL_ERROR")))
		if err != nil {
			fmt.Println("Write data failed:", err.Error())
			os.Exit(1)
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
			serversSlice = append(serversSlice, connection{host: host, port: port, con_type: "tcp"})
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
