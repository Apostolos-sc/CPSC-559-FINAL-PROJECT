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

var master_server_index = 0
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
	gameRooms     = make(map[string]*gameRoom)
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

func restartServer() *net.TCPConn {
	//Call the new designated master server to load all game rooms
	log.Printf("Attempting to call Back Up server to reestablish connections.")
	var err error
	master_server_index = (master_server_index + 1) % (len(serverList))
	log.Printf("Master server index is now %d.", master_server_index)
	serverMutex.Lock()
	server_addr, err := net.ResolveTCPAddr("tcp", serverList[master_server_index].host+":"+serverList[master_server_index].port)
	serverMutex.Unlock()
	var gameRoomConn *net.TCPConn = nil
	if len(gameRooms) > 0 {
		for key, value := range gameRooms {
			log.Printf("Inside mutex, calling backup server with IP address %s:%s.", serverList[master_server_index].host, serverList[master_server_index].port)
			gameRoomConn, err = net.DialTCP("tcp", nil, server_addr)
			if err != nil {
				log.Printf("Failed to create a new tcp connection for game Room %s", key)
			} else {
				_, err = gameRoomConn.Write([]byte("Restart Room Connection:" + key + ":"))
				if err != nil {
					log.Printf("Failed to send USER_IN_ROOM_ALREADY_ERROR. %s\n", err.Error())
				}
				//We need to close previous connection -> The next line will need testing
				//value.gameRoomConn.Close()
				log.Printf("Previous server serving game Room : %s is %s.", key, value.gameRoomConn.RemoteAddr())
				gameRooms[key].gameRoomConn = gameRoomConn
				log.Printf("Successfully Reconnected game Room %s to server with addres %s.", key, gameRooms[key].gameRoomConn.RemoteAddr())
			}
		}
	} else {
		log.Printf("There are no game rooms. No connection reestablishments needed.")
	}
	return gameRoomConn
}
