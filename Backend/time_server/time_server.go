package main

import (
	"github.com/gorilla/websocket"
	"log"
	"strconv"
	"strings"
	"time"
	"fmt"
)

func main() {
	var portRead = -5
	log.Printf("Please give the port number that the server will be servicing Application Servers on (between 6600 and 6620) :.\n")
	_, scanErr := fmt.Scan(&portRead)
	for scanErr != nil || portRead < 6600 || portRead > 6620 {
		if scanErr == nil {
			log.Printf("Port number for Application Server Registration must be between 6600 and 6620.\n")
		} else {
			log.Print("Scan for port failed, due to error : ", scanErr.Error())
		}
		_, scanErr = fmt.Scan(&portRead)
		log.Printf("Please give the port number that the server will be servicing Application Servers on (between 6600 and 6620):.\n")
	}
	SERVER_REGISTRATION.port = strconv.Itoa(portRead)
	log.Printf("Read Server Registration Port # : %d.\n", portRead)
	portRead = -5
	log.Printf("Please give the port number that the server will be servicing Clients on (between 7000 and 7010) :.\n")
	_, scanErr = fmt.Scan(&portRead)
	for scanErr != nil || portRead < 7000 || portRead > 7010 {
		if scanErr == nil {
			log.Printf("Port number for Client Registration must be between 7000 and 7010.\n")
		} else {
			log.Print("Scan for port failed, due to error : ", scanErr.Error())
		}
		_, scanErr = fmt.Scan(&portRead)
		log.Printf("Please give the port number that the server will be servicing Clients on (between 6600 and 6620):.\n")
	}
	CLIENT_SERVICE.port = strconv.Itoa(portRead)
	log.Printf("Read Client Registration Port # : %d.\n", portRead)
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
	// Add client to a gameroom based on client websocket connection
	for {
		_, buffer, err = clientConn.ReadMessage()
		if err != nil {
		    log.Printf("Client Sent Message : %s.", string(buffer[:len(buffer)]))
			if strings.Compare(username, "") == 0 {
				//Player disconnected before making a request to the server
				// Remove the client if disconnection detected
				log.Println("Failed to read a request from the client. Connection will be terminated.")
				clientConn.Close()
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
				gameRoomMutex.Lock()
				_, ok := gameRooms[command[1]]
				if ok {
					gameRooms[command[1]].players[command[2]] = clientConn
					gameRoomMutex.Unlock()
					log.Printf("Player %s from room %s, successfully joined the time server. ", username, accessCode)
				} else {
					log.Printf("Player %s cannot register to game Room %s because it doesn't exist.", username, accessCode)
					clientConn.Close()
					gameRoomMutex.Unlock()
					break
				}
				
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
		if err != nil {
			log.Printf("There was an issue with reading from the potential Server with IP %s. Error : %s.\n", conn.RemoteAddr(), err.Error())
			log.Printf("Connection will be terminated.")
			conn.Close()
			break
		}
		n = len(buffer)
		log.Printf("The buffer is %s", buffer)
		log.Printf("the string buffer is %s", "_"+string(buffer[:n])+"_")

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
			err = conn.WriteMessage(1, []byte("Accepted"))
			if err != nil {
			    log.Printf("Error when sending response : %s", err.Error())
			}
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
		} else if strings.Compare(command[0], "Create Room") == 0 {
		    log.Printf("hello")
			// Protocol Game Over:accesscode
			gameRoomMutex.Lock()
			_, ok := gameRooms[command[1]]
			if !ok {
				//Check to make sure that the game room was not already created, this can happen only if the server crashed before the Ready command was fully completed
				//ie, after the Create Room instruction and before sending that all Players are ready to proxy
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
			log.Printf("Invalid command by the server with IP address %s. Connection will now be terminated.", conn.RemoteAddr())
			conn.Close()
			break
		}
	}
}
