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

//Connection informatio
type connection struct {
	host string;
	port string;
	con_type string;
}
//gameRoom ID and the connection that will service it
type gameRoom struct {
	ID int;
	server net.Conn;
}
//Address to be listening for servers to indicate they want to serve
var SERVER_REGISTRATION = connection {"127.0.0.1", "9000", "tcp"}
//Address to be listening for clients
var CLIENT_SERVICE = connection {"127.0.0.1", "8000", "tcp"}
//Will be used to keep track of servers that are servicing
var (
	mu sync.Mutex
	servers_slice[] connection
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

func handleClientRequest(conn net.Conn) {
	//Handle Client Request Here
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
		host, port, err = net.SplitHostPort(string(buffer[:n]))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("")
			mu.Lock()
			servers_slice = append(servers_slice, connection{host:host, port:port, con_type:"tcp"})
			mu.Unlock()
			fmt.Printf("%s was added as a server on the server list on %v.\n", string(buffer[:n]), time)
			conn.Write([]byte("Received Address"))
		}
	} else {
		conn.Write([]byte("Your registration as a server is declined."))
	}
	// close conn
	conn.Close()
}