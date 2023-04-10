package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	go serverListener()
	clientListener()
}

func handleClientRequest(clientConn *websocket.Conn, connID int) {
	var n int
	buffer := make([]byte, 1024)
	err = clientConn.WriteMessage(1, []byte("Connection Established"))
	if err != nil {
		log.Println("Failed to send a message to the client. Connection will be terminated.")
	} else {
		log.Printf("Message send to the client with ID : %d.\n", connID)
	}
	// Add client to a gameroom based on client websocket connection
	for {
		_, buffer, err = clientConn.ReadMessage()
		if err != nil {
			log.Println("Failed to read a request from the client. Connection will be terminated.")
			break //exit the loop here
		} else {
			log.Printf("Message received from client with ID : %d.\n", connID)
			n = len(buffer)
			var command = strings.Split(string(buffer[:n]), ":")
			// Still need to implement get timer
			if strings.Compare(command[0], "ClientJoin") == 0 {
				// format of message: ClientJoin:accessCode:username
				// Need to check if the right person is making the request
				gameRooms[command[1]] = make(map[string]*websocket.Conn)
				gameRooms[command[1]][command[2]] = clientConn
			}
		}
	}
}

// Need to make it such that it continuously listens for the server request
func handleServerRegistration(conn net.TCPConn) {
	var buffer = make([]byte, 1024)
	_, err = conn.Read(buffer)
	if err != nil {
		log.Printf("There was an issue with reading from the potential Server with IP %s. Error : %s.\n", conn.RemoteAddr(), err.Error())
	}
	// Steps:
	// 1. Read for the server message
	// 2. Check the message protocol
	// 3. Send Accepted if timeserver request was accepted
	var n = len(buffer)
	var command = strings.Split(string(buffer[:n]), ":")
	if strings.Compare(string(buffer[:n]), "Server Join") == 0 {
		//Client attempting to connect is a server
		//host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			log.Printf("There was an error while splitting the remote Address of server. Error : %s.", err.Error())
		}
		log.Printf("Command : %v. Send Accepted.\n", string(buffer[:]))
		conn.Write([]byte("Accepted"))

	} else if strings.Compare(command[0], "StartTimer") == 0 {
		// Format of message: StartTimer:accessCode:round
		// Starts the timer for a game room -> server sends
		// Starts sending timer every one sec to all the clients -> need list of clients for each gameroom
		// End the timer if 30 secs have passed.
		ticker := time.NewTicker(1 * time.Second)
		done := make(chan bool)
		go func() {
			for {
				select {
				case t := <-ticker.C:
					for _, value := range gameRooms[command[1]] {
						err = value.WriteMessage(1, []byte(t.String()))
						if err != nil {
							//log.Printf("There was an issue while sending All Ready acknowledgment to player %s with IP address : %s. Error : %s \n", key, value.RemoteAddr(), err.Error())
						} else {
							log.Printf("Error sending timer to client")
						}
					}
				case <-done:
					ticker.Stop()
				}
			}
		}()
		time.Sleep(31 * time.Second) // Stopping the timer at the end of 31 secs
		ticker.Stop()
		done <- true

	} else {
		_, err = conn.Write([]byte("Wrong command given, access declined."))
		if err != nil {
			log.Printf("There was an issue while informing the potential server that the given command is incorrect. \n %s", err.Error())
		}
	}
	// close conn
	conn.Close()
}
