package main

import (
	"github.com/gorilla/websocket"
	"log"
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
// 	err = clientConn.WriteMessage(1, []byte("Time Server: Connection Established"))
// 	if err != nil {
// 		log.Println("Failed to send a message to the client. Time Server Connection will be terminated.")
// 	} else {
// 		log.Printf("Message send to the client with ID : %d.\n", connID)
// 	}
	// Add client to a gameroom based on client websocket connection
	for {
		_, buffer, err = clientConn.ReadMessage()
		if err != nil {
		    log.Print("Client Sent Message : %s.", string(buffer[:len(buffer)]))
			if strings.Compare(username, "") == 0 {
				//Player disconnected before making a request to the server
				// Remove the client if disconnection detected
				log.Println("Failed to read a request from the client. Connection will be terminated.")
			} else {
				//We need to remove from gameMap
				//lock the specific room
				log.Printf("Attempting to delete username with %s from gameRoom : %s", username, accessCode)
				for key, value := range gameRooms {
				    for username, _ := range value.players {
				        log.Printf("Player : %s in room : %s", key, username)
				    }
				}
				log.Printf("Game Rooms Map %v", gameRooms)
				gameRooms[accessCode].localMutex.Lock()
				delete(gameRooms[accessCode].players, username)
				log.Printf("Successfully removed player %s of game Room %s from time server.", username, accessCode)
				gameRooms[accessCode].localMutex.Unlock()
				if len(gameRooms[accessCode].players) == 0 {
					//Lock the gameRooms and remove the remove, last player disconnected
					gameRoomMutex.Lock()
					gameRooms[accessCode].ticker.Stop()
					log.Printf("Ticker for game room %s has been terminated.", accessCode)
					delete(gameRooms, accessCode)
					log.Printf("Game Room with code %s is not being tracked by time server anymore!", accessCode)
					gameRoomMutex.Unlock()
					break
				}
			}
			break //exit the loop here
		} else {
			log.Printf("Message received from client with ID : %d. Request : %s\n", connID, string(buffer[:len(buffer)]))
			n = len(buffer)
			var command = strings.Split(string(buffer[:n]), ":")
			if strings.Compare(command[0], "Client Join") == 0 {
			    username = command[2]
			    accessCode = command[1]
				// format of message: Client Join:accessCode:username
                gameRooms[command[1]].localMutex.Lock()
                gameRooms[command[1]].players[command[2]] = clientConn
				gameRooms[command[1]].localMutex.Unlock()
				log.Printf("Player %s from room %s, successfully joined the time server. ", username, accessCode)
			} else {
				log.Printf("Unrecognized message format from the " +
					"client, terminating connection")
				break
			}
		}
	}
}

// Need to make it such that it continuously listens for the server request
func handleServerRegistration(conn *websocket.Conn) {
	log.Printf("In handle registration")
	var buffer = make([]byte, 1024)
	var n int
	var err error
	for {

		_, buffer, err = conn.ReadMessage()
		n = len(buffer)
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
			conn.WriteMessage(1, []byte("Accepted"))

		} else if strings.Compare(command[0], "Start Timer") == 0 {
			// Strat Timer:accesscode:round
			if gameRooms[command[1]].ticker != nil {
			    gameRooms[command[1]].ticker.Stop()
			}
			gameRooms[command[1]].ticker = time.NewTicker(1*time.Second)
			start := allowed_time
			done := make(chan bool)
			go func() {
				for {
					select {
					case <-gameRooms[command[1]].ticker.C:
						start--
						log.Printf("Game Room: %s, Timer : %s", command[1], strconv.Itoa(start))
						for _, value := range gameRooms[command[1]].players {
							err = value.WriteMessage(1, []byte("Time:"+strconv.Itoa(start)))
							//log.Printf("the time is %s",t.String())
							if err != nil {
                                log.Printf("Error sending timer to clients from time server")
							}
						}
                        if start <= 0 {
                            gameRooms[command[1]].ticker.Stop()
                            done <- true
                        }
 					case <-done:
 						gameRooms[command[1]].ticker.Stop()
					}
				}
			}()
// 			time.Sleep(time.Duration(allowed_time) * time.Second) // Stopping the timer at the end of 31 secs
// 			ticker.Stop()
// 			done <- true
		} else if strings.Compare(command[0], "Create Room") == 0 {
		    log.Printf("hello")
			// Protocol Game Over:accesscode
			gameRoomMutex.Lock()
			_, ok := gameRooms[command[1]]
			if !ok {
			    gameRooms[command[1]] = &gameRoom {currentRound:0, players: make(map[string]*websocket.Conn), ticker: nil}
			    log.Printf("Creating game Room")
			}
			gameRoomMutex.Unlock()
		} else if strings.Compare(command[0], "Game Over") == 0 {
            // Protocol Game Over:accesscode
            gameRoomMutex.Lock()
            _, ok := gameRooms[command[1]]
            if ok {
                delete(gameRooms, command[1])
            }
            gameRoomMutex.Unlock()
        }else {
			log.Printf("Invalid command by the server")
			err = conn.WriteMessage(1, []byte("Wrong command given, access declined - Time server."))
			if err != nil {
				log.Printf("Time server connection failed unexpectedly %s", err.Error())
				conn.Close()
				break
			}
		}
	}
}
