package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func main() {
	log.Printf("In main")
	go serverListener()
	log.Printf("after server listener started")
	clientListener()
}

func handleClientRequest(clientConn *websocket.Conn, connID int) {
	var accessCode string = ""
	var username string = ""
	log.Printf("In handle client request")
	var n int
	buffer := make([]byte, 1024)
	err = clientConn.WriteMessage(1, []byte("Time Server: Connection Established"))
	if err != nil {
		log.Println("Failed to send a message to the client. Time Server Connection will be terminated.")
	} else {
		log.Printf("Message send to the client with ID : %d.\n", connID)
	}
	// Add client to a gameroom based on client websocket connection
	for {
		_, buffer, err = clientConn.ReadMessage()
		if err != nil {
			if strings.Compare(username, "") == 0 {
				//Player disconnected before making a request to the server
				// Remove the client if disconnection detected
				log.Println("Failed to read a request from the client. Connection will be terminated.")
			} else {
				//We need to remove from gameMap
				//lock the specific room
				gameRooms[accessCode].Lock()
				delete(gameRooms[accessCode].players, username)
				gameRooms[accessCode].Unlock()
				if len(gameRooms[accessCode].players) == 0 {
					//Lock the gameRooms and remove the remove, last player disconnected
					gameRoomMutex.Lock()
					delete(gameRooms, accessCode)
					gameRoomMutex.Unlock()
				}
			}
			break //exit the loop here
		} else {
			log.Printf("Message received from client with ID : %d.\n", connID)
			n = len(buffer)
			var command = strings.Split(string(buffer[:n]), ":")
			if strings.Compare(command[0], "Client Join") == 0 {
				// format of message: Client Join:accessCode:username
				// Need to check if the right person is making the request
				gameRoomMutex.Lock()
				//if it doesn't exist it probably was deleted before previous server crashed ??
				_, ok := gameRooms[command[1]]
				if ok {
					gameRooms[command[1]].Lock()
					for _, value := range gameRooms[command[1]].players {
						value.Close()
					}
				}
				delete(gameRooms, command[1])
				gameRoomMutex.Unlock()
			} else {
				log.Printf("Unrecognized message format from the " +
					"client, terminating connection")
				break
			}
		}
	}
}

// Need to make it such that it continuously listens for the server request
func handleServerRegistration(conn net.Conn) {
	log.Printf("In handle registration")
	var buffer = make([]byte, 1024)
	for {

		n, err := conn.Read(buffer)
		log.Printf("The buffer is %s", buffer)
		log.Printf("the string buffer is %s", "_"+string(buffer[:n])+"_")
		if err != nil {
			log.Printf("There was an issue with reading from the potential Server with IP %s. Error : %s.\n", conn.RemoteAddr(), err.Error())
		}
		var command = strings.Split(string(buffer[:n]), ":")
		log.Printf("command [0] is %s", command[0])
		log.Printf("command is %s", command)
		if strings.Compare(string(buffer[:n]), "Server Join") == 0 {
			//Client attempting to connect is a server
			//host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
			if err != nil {
				log.Printf("There was an error while splitting "+
					"the remote Address of server in Time Server, Error : %s.", err.Error())
			}
			log.Printf("Command : %v. Send Accepted.\n", string(buffer[:]))
			conn.Write([]byte("Accepted"))

		} else if strings.Compare(command[0], "Start Timer") == 0 {
			// Strat Timer:accesscode:round
			ticker := time.NewTicker(1 * time.Second)
			start := allowed_time
			done := make(chan bool)
			go func() {
				for {
					select {
					case <-ticker.C:
						start--
						log.Printf(strconv.Itoa(start))
						for _, value := range gameRooms[command[1]].players {
							err = value.WriteMessage(1, []byte(strconv.Itoa(start)))
							//log.Printf("the time is %s",t.String())
							if err != nil {
							} else {
								log.Printf("Error sending timer to server from time server")
							}
						}
					case <-done:
						ticker.Stop()
					}
				}
			}()
			time.Sleep(time.Duration(allowed_time) * time.Second) // Stopping the timer at the end of 31 secs
			ticker.Stop()
			done <- true

		} else if strings.Compare(command[0], "Game Over") == 0 {
			// Protocol Game Over:accesscode
			gameRoomMutex.Lock()
			_, ok := gameRooms[command[1]]
			if ok {
				delete(gameRooms, command[1])
			} else {
				gameRooms[command[1]].players = make(map[string]*websocket.Conn)
				gameRooms[command[1]].players[command[2]] = clientConn
			}
			gameRoomMutex.Unlock()
		} else {
			log.Printf("Invalid command by the server")
			_, err = conn.Write([]byte("Wrong command given, access declined - Time server."))
			if err != nil {
				log.Printf("Time server connection failed unexpectedly %s", err.Error())
				conn.Close()
				break
			}
		}
	}
}
