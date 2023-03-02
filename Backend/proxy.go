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
	accessCode string;
	server connection;
}

//Maximum number of game rooms a server should handle
var MAX_ROOMS_PER_SERVER int = 2
//Address to be listening for servers to indicate they want to serve
var SERVER_REGISTRATION = connection {"127.0.0.1", "9000", "tcp"}
//Address to be listening for clients
var CLIENT_SERVICE = connection {"127.0.0.1", "8000", "tcp"}
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
	gameRoomsSlice[] gameRoom
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
		//Server Clients
		conn, err:= listener.Accept()
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
			os.Exit(1)
		}
		fmt.Printf("Potential Server Registration Request Incoming from : %s\n", conn.RemoteAddr().String())
		go handleServerRegistration(conn)
	}
}
//handle 
func handleClientRequest(conn net.Conn) {
	//Handle Client Request Here
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	if strings.Compare(string(buffer[:n]), "Create Room") == 0 {
		//Client wants to create a game room - Handle game room
		
	}
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