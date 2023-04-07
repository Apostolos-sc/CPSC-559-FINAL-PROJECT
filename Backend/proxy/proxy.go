package main

//Proxy
import (
	"github.com/gorilla/websocket"
	"log"
	"net"
	"strings"
	"sync"
)

// Connection information
type connection struct {
	host     string
	port     string
	con_type string
}

// gameRoom with an access Code and a server that will serve
type gameRoom struct {
	gameRoomConn *net.TCPConn
	players      map[string]*websocket.Conn
}

var client_counter = 0
var ip_address = "10.0.0.105"

// SERVER_REGISTRATION Address to be listening for servers to indicate they want to serve
var SERVER_REGISTRATION = connection{ip_address, "9000", "tcp"}

// CLIENT_SERVICE Address to be listening for clients
var CLIENT_SERVICE = connection{ip_address, "8000", "tcp"}

// Will be used to keep track of servers that are servicing
// make hashmap here when back from soccer
var (
	serverMutex sync.Mutex
	serverList  []connection
)

// Will be used to keep track of the gameRooms being serviced
var (
	gameRoomMutex sync.Mutex
	gameRooms     = make(map[string]gameRoom)
)

func main() {
	go serverListener()
	clientListener()
}

func checkRequest(command []string) bool {
	log.Printf("Inside check Request.\n")
	if len(command) == 2 || len(command) == 3 || len(command) == 4 {
		if len(command) == 2 {
			if strings.Compare(command[0], "Create Room") == 0 {
				return true
			} else {
				return false
			}
		} else if len(command) == 3 {
			if strings.Compare(command[0], "Join Room") == 0 || strings.Compare(command[0], "Start Game") == 0 || strings.Compare(command[0], "Stop Game") == 0 || strings.Compare(command[0], "Ready") == 0 {
				return true
			} else {
				return false
			}
		} else {
			if strings.Compare(command[0], "Answer") == 0 {
				return true
			} else {
				return false
			}
		}
	} else {
		return false
	}
}
