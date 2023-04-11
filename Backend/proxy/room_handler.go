package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func handleClientRequest(clientConn *websocket.Conn, connID int) {
     var timeout = 5*time.Second
//     SetReadDeadline(time.Now().Add(timeout))
// SetReadDeadline(time.Now().Add(time.Time{})
	var time_stamp int64
	var service_completed = true
	var command []string
	var request string
	var gameRoomConn *net.TCPConn = nil
	var keepServicing = true
	var err error
	// Number of bytes in the buffer
	var n int
	var nResponse int
	buffer := make([]byte, 1024)
	//var serviceComplete bool = false
	var disconnect = false
	var previousCommand = "None"
	serverResponseBuffer := make([]byte, 1024)
	err = clientConn.WriteMessage(1, []byte("Connection Established"))
	if err != nil {
		log.Println("Failed to send a message to the client. Connection will be terminated.")
		log.Printf("Previous command : %s.\n", previousCommand)
		disconnect = true
	} else {
		log.Printf("Message send to the client with ID : %d.\n", connID)
	}
	for {
		if !keepServicing {
			log.Printf("Client cannot be reached. Connection will be terminated.\n")
			break
		}
		if disconnect {
			if strings.Compare(previousCommand, "None") == 0 {
				//no need to do anything
				log.Print("No previous activity detected. Connection will be terminated.\n")
			} else {
				// Command format = command:username:accesscode
				//For some bizarre reason the software adds a "." at the end of response[1] when I append it to previous command
				log.Printf("There previous raw command is : %s\n", previousCommand)
				var previousCommandString = strings.Split(previousCommand, ":")
				log.Printf("The command length is : %d.\n", len(previousCommandString))
				log.Printf("The previous command is : %s.\n", previousCommand)
				log.Printf("The game Room is %s.\n", previousCommandString[2])
				gameRoomMutex.Lock()
				_, ok := gameRooms[previousCommandString[2]]
				if !ok {
					log.Printf("The game connection for room %s is no longer available.\n", previousCommandString[2])
				} else {
					if len(gameRooms[previousCommandString[2]].players) == 1 {
						//if it's one player in the room the game will end
						log.Printf("There is only one player in the room, let's contact the server, delete the user and the room.\n")
						if gameRoomConn == nil {
							gameRoomConn = gameRooms[previousCommandString[2]].gameRoomConn
						}
						//log.Printf("User %s, is trying to disconnect from room %s", username, accessCode)
						gameRoomConn.SetWriteDeadline(time.Now().Add(timeout))
						_, err := gameRoomConn.Write([]byte("Disconnect:" + previousCommandString[1] + ":" + previousCommandString[2] + ":" + strconv.FormatInt(time_stamp, 10)))
						if err != nil {
							log.Printf("Sending the following Disconnect information to the server \"Discconect:%s:%s to server\" %s failed. Error : %s\n", previousCommandString[1], previousCommandString[2], gameRoomConn.RemoteAddr(), err.Error())
							gameRooms[previousCommandString[2]].gameRoomConn.SetWriteDeadline(time.Time{})
							gameRoomConn = restartServer()
							gameRoomMutex.Unlock()
							continue
						}
						gameRooms[previousCommandString[2]].gameRoomConn.SetWriteDeadline(time.Time{})
						gameRooms[previousCommandString[2]].gameRoomConn.SetReadDeadline(time.Now().Add(timeout))
						nResponse, err = gameRooms[previousCommandString[2]].gameRoomConn.Read(serverResponseBuffer)
						if err != nil {
							//Operation failed, let's wake up back up server
							log.Printf("The server failed to inform us that the game was successfully terminated. Find new server.")
							gameRooms[previousCommandString[2]].gameRoomConn.SetReadDeadline(time.Time{})
							gameRoomConn = restartServer()
							gameRoomMutex.Unlock()
							continue
						} else {
							//We know operation completed successfully
							gameRooms[previousCommandString[2]].gameRoomConn.SetReadDeadline(time.Time{})
							log.Printf("Server sent : %s", string(serverResponseBuffer[:nResponse]))
						}
						delete(gameRooms[previousCommandString[2]].players, previousCommandString[1])
						gameRoomConn.Close()
						delete(gameRooms, previousCommandString[2])
						gameRoomMutex.Unlock()
						log.Printf("Game Room : %s has stopped servicing players.\n", previousCommandString[2])
						break
					} else {
						log.Printf("There are more players in the room{%d}, let's contact the server, disconnect the user: %s\n", len(gameRooms[previousCommandString[2]].players), previousCommandString[1])
						// The game room can be null when the previousCommand is not CreateGameRoom but JoinRoom
						// Test this, added on line 397
						if gameRoomConn == nil {
							gameRoomConn = gameRooms[previousCommandString[2]].gameRoomConn
						}
						gameRoomConn.SetWriteDeadline(time.Now().Add(timeout))
						_, err := gameRoomConn.Write([]byte("Disconnect:" + previousCommandString[1] + ":" + previousCommandString[2] + ":" + strconv.FormatInt(time_stamp, 10)))
						if err != nil {
						    gameRoomConn.SetWriteDeadline(time.Time{})
							//Server was unreachable when attempting to send disconnect inform
							log.Printf("Sending the following Disconnect information to the server \"Discconect:%s:%s to server\" %s failed. Error : %s\n", previousCommandString[1], previousCommandString[2], gameRoomConn.RemoteAddr(), err.Error())
							gameRoomConn = restartServer()
							gameRoomMutex.Unlock()
							continue
						}
						gameRoomConn.SetWriteDeadline(time.Time{})
						gameRoomConn.SetReadDeadline(time.Now().Add(timeout))
						nResponse, err = gameRooms[previousCommandString[2]].gameRoomConn.Read(serverResponseBuffer)
						if err != nil {
							log.Printf("There was an error while waiting for the server to respond on disconnect command. Error : %s\n", err.Error())
							gameRoomConn.SetReadDeadline(time.Time{})
							gameRoomConn = restartServer()
							gameRoomMutex.Unlock()
							service_completed = false
							continue
						} else {
						    gameRoomConn.SetReadDeadline(time.Time{})
							delete(gameRooms[previousCommandString[2]].players, previousCommandString[1])
							log.Printf("The server responded %s after player disconnected.\n", string(serverResponseBuffer[:nResponse]))
							var disconnectResponse = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
							//Happens if the last player that disconnected was the only one that wasn't ready
							if strings.Compare(disconnectResponse[0], "All Ready") == 0 {
								// Inform all players that the player disconnected from the lobby
								for key, value := range gameRooms[previousCommandString[2]].players {
									err = value.WriteMessage(1, []byte("Lobby Disconnect:"+previousCommandString[1]))
									if err != nil {
										log.Printf("There was an issue while sending Lobby Disconnect:%s acknowledgment to player %s with IP address : %s. Error : %s \n", previousCommandString[1], key, value.RemoteAddr(), err.Error())
										//We don't care, the client's own game routine needs to handle this.
									}
								}
								// Inform all the players that it is time to start the game{all players except for the one that disconnected were ready}
								for key, value := range gameRooms[previousCommandString[2]].players {
									err = value.WriteMessage(1, serverResponseBuffer[:nResponse])
									if err != nil {
										log.Printf("There was an issue while sending All Ready acknowledgment to player %s with IP address : %s. Error : %s \n", key, value.RemoteAddr(), err.Error())
										//We don't care, the client's own game routine needs to handle this.
									}
								}
							} else if strings.Compare(disconnectResponse[0], "Lobby Disconnect") == 0 {
								//Let's inform the other clients that the user disconnected.
								for key, value := range gameRooms[previousCommandString[2]].players {
									err = value.WriteMessage(1, serverResponseBuffer[:nResponse])
									if err != nil {
										log.Printf("There was an issue while sending Lobby Disconnect acknowledgment to player %s with IP address : %s. Error : %s \n", key, value.RemoteAddr(), err.Error())
										//We don't care, the client's own game routine needs to handle this.
									}
								}
							} else if strings.Compare(disconnectResponse[0], "Everyone Responded") == 0 {
								//Send to all connected players that everyone has responded, and the question to start next round
								for key, value := range gameRooms[previousCommandString[2]].players {
									err = value.WriteMessage(1, []byte(serverResponseBuffer[:nResponse]))
									if err != nil {
										log.Printf("There was an issue while sending Everyone Responded acknowledgment to Player %s with IP address : %s. Error : %s. \n", key, value.RemoteAddr(), err.Error())
										//Client that we are sending the everyone responded message disconnected, its own subroutine will handle this
									} else {
										log.Printf("Sent Everyone Responded to player %s with IP address %s.\n", key, value.RemoteAddr())
									}
								}
							} else if strings.Compare(disconnectResponse[0], "Game Over") == 0 {
								//Send to all players that are still connected that the game is over
								for key, value := range gameRooms[previousCommandString[2]].players {
									err = value.WriteMessage(1, []byte(serverResponseBuffer[:nResponse]))
									if err != nil {
										log.Printf("There was an issue while sending Everyone Responded acknowledgment to Player %s with IP address : %s. Error : %s. \n", key, value.RemoteAddr(), err.Error())
										//Client that we are sending the everyone responded message disconnected, its own subroutine will handle this
									} else {
										log.Printf("Sent Everyone Responded to player %s with IP address %s.\n", key, value.RemoteAddr())
									}
								}
							}
						}
					}
				}
				gameRoomMutex.Unlock()
			}
			break
		}

		if service_completed == true {
			log.Printf("Waiting to read from client with ID %d.\n", connID)
			_, buffer, err = clientConn.ReadMessage()
			if err != nil {
				log.Println("Failed to read a request from the client. Connection will be terminated.")
				log.Printf("Previous command : %s.\n", previousCommand)
				disconnect = true
				continue
			} else {
				log.Printf("Message received from client with ID : %d.\n", connID)
				//create time stamp for command we will send to server
				now := time.Now()
				time_stamp = now.UnixNano()
			}
			//Assign this command as the one that was executed before
			n = len(buffer)
			previousCommand = string(buffer[:len(buffer)]) + ":" + strconv.FormatInt(time_stamp, 10)
			log.Printf("Previous command set as :%s", previousCommand)
			//critical access
			serverMutex.Lock()
			if len(serverList) == 0 {
				//Can't service Client, no live Servers.
				log.Println("There are no servers available to service Clients. Send Error to Client. Connection Terminated.")
				err = clientConn.WriteMessage(1, ([]byte("Error:No Server Available")))
				if err != nil {
					log.Println("Unable to send to client Error: No Server Available:", err.Error())
				}
				serverMutex.Unlock()
				keepServicing = false
				continue
			}
			serverMutex.Unlock()
			command = strings.Split(string(buffer[:n]), ":")
			request = string(buffer[:n])
		}
		//check if proxy received a valid request
		if checkRequest(command) {
			if strings.Compare(command[0], "Create Room") == 0 {
				//Critical Section
				gameRoomMutex.Lock()
				gameRoomAddr, err := net.ResolveTCPAddr("tcp", serverList[master_server_index].host+":"+serverList[master_server_index].port)
				address := serverList[master_server_index].host + ":" + serverList[master_server_index].port
				if err != nil {
					log.Printf("ResolveTCPAddr failed for %s:%v\n", address, err.Error())
					//what happens if can't resolve?
				}
				log.Printf("Create Room Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				//attempt to connect to server to establish a game room connection using a tcp
				gameRoomConn, err = net.DialTCP("tcp", nil, gameRoomAddr)
				if err != nil {
					//innaccessible server when dialing, let's load data in a secondary server
					log.Printf("Dialing server %s for game Room Creation Request failed: %v\n", address, err.Error())
					gameRoomConn = restartServer()
					gameRoomMutex.Unlock()
					service_completed = false
					continue
				}
				//forward gameRoom creation request to server
				log.Printf("Requesting from Server with address %s Game Room Creation.\n", gameRoomConn.RemoteAddr().String())
				gameRoomConn.SetWriteDeadline(time.Now().Add(timeout))
				_, err = gameRoomConn.Write([]byte(string(buffer[:n]) + ":" + strconv.FormatInt(time_stamp, 10)))
				if err != nil {
					//Server crashed after we dialed, let's pick another one
					log.Printf("Sending the Game Room Creation Command : %s to the server failed: %s", string(buffer[:n]), err.Error())
					gameRoomConn.SetWriteDeadline(time.Time{})
					gameRoomConn = restartServer()
					gameRoomMutex.Unlock()
					service_completed = false
					continue
				}
				gameRoomConn.SetWriteDeadline(time.Time{})
				//wait for response
				gameRoomConn.SetReadDeadline(time.Now().Add(timeout))
				nResponse, err = gameRoomConn.Read(serverResponseBuffer)
				if err != nil {
					//server crashed while we were waiting for it to respond to our request
					log.Println("Read data failed:", err.Error())
					gameRoomConn.SetReadDeadline(time.Time{})
					log.Printf("Failed to receive a response from server : %s. Will attempt to connect to another server.\n", gameRoomConn.RemoteAddr().String())
					gameRoomConn = restartServer()
					gameRoomMutex.Unlock()
					service_completed = false
					continue
				}
				gameRoomConn.SetReadDeadline(time.Time{})
				var response = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
				if strings.Compare(response[0], "Room Created") == 0 {
					//Server allows creation of room
					log.Printf("Game Room successfully created. Access Code : %s and is served by : %s\n", response[1], gameRoomConn.RemoteAddr().String())
					//Initialiaze game room struct and assign to map: gameRooms[accessCode] = gameRoom
					gameRooms[response[1]] = &gameRoom{gameRoomConn, make(map[string]*websocket.Conn)}
					//player should be added to the player hashmap in the gameRoom with accessCode : response[1]
					gameRooms[response[1]].players[command[1]] = clientConn
					//Come Back to Create Game Room, Potential Race Conditions, add timestamps
					// string(buffer[:n]) = Create Room:Username and response[1] = accessCode
					previousCommand = string(buffer[:n]) + ":" + response[1]
					log.Printf("Previous command set as :%s.\n", previousCommand)
					err = clientConn.WriteMessage(1, []byte("Access Code:"+response[1]))
					if err != nil {
						log.Printf("Write Access Code:%s to client %s failed: %s\n", response[1], clientConn.RemoteAddr().String(), err.Error())
						disconnect = true
						gameRoomMutex.Unlock()
						continue
					} else {
						// Successful sent to user
						log.Printf("Client with username %s & IP address %s received the Access Code.\n", command[1], clientConn.RemoteAddr().String())
					}
				} else if strings.Compare(response[0], "ROOM_CREATION_ERROR") == 0 {
					//Room couldn't be created inform client.
					err = clientConn.WriteMessage(1, []byte("Error:Room Creation"))
					if err != nil {
						//Client can't receive the message. Not in a room we don't care.
						log.Printf("Unable to send ROOM_CREATION_ERROR to client %s: %s\n", clientConn.RemoteAddr().String(), err.Error())
						keepServicing = false
						break
					}
				} else if strings.Compare(response[0], "USER_IN_ROOM_ALREADY_ERROR") == 0 {
					//User already in another room, bye bye
					err = clientConn.WriteMessage(1, []byte("Error:User In Room Already"))
					if err != nil {
						log.Printf("Unable to send USER_IN_ROOM_ALREADY_ERROR to client %s: %s\n", clientConn.RemoteAddr().String(), err.Error())
						//we don't care if the user disconnects at this point.
						keepServicing = false
					}
				} else {
					//Corrupt message? Or timeout handling here - figure it out later some recovery
					log.Printf("Server Response for Command : %s was %s.\n", string(buffer[:n]), string(serverResponseBuffer[:nResponse]))
				}
				service_completed = true
				gameRoomMutex.Unlock()
			} else if strings.Compare(command[0], "Join Room") == 0 {
				//First let's check if the room exists - Lock Critical Resource first - We need this to redirect the traffic
				log.Printf("Join Room Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				gameRoomMutex.Lock()
				_, ok := gameRooms[command[2]]
				//The Game Room exists
				if ok == true {
					log.Printf("Requesting from Server with address %s a Join Room Request.\n", gameRooms[command[2]].gameRoomConn.RemoteAddr().String())
					gameRooms[command[2]].gameRoomConn.SetWriteDeadline(time.Now().Add(timeout))
					_, err = gameRooms[command[2]].gameRoomConn.Write([]byte(string(buffer[:n]) + ":" + strconv.FormatInt(time_stamp, 10)))
					if err != nil {
						//Contact another server
						log.Printf("Sending Join Room Command %s to the server of game Room %s failed.\n", request, command[2])
						gameRooms[command[2]].gameRoomConn.SetWriteDeadline(time.Time{})
						gameRoomConn = restartServer()
						gameRoomMutex.Unlock()
						service_completed = false
						continue
					}
					gameRooms[command[2]].gameRoomConn.SetWriteDeadline(time.Time{})
					gameRooms[command[2]].gameRoomConn.SetReadDeadline(time.Now().Add(timeout))
					nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
					if err != nil {
						//failed to receive response for join room
						log.Printf("Receiving Response from Server for Join Room Failed. Let's connect to another server.")
						//Serve Crashed when receiving response, connect to another server.
						gameRooms[command[2]].gameRoomConn.SetReadDeadline(time.Time{})
						gameRoomConn = restartServer()
						gameRoomMutex.Unlock()
						service_completed = false
						continue
					} else {
						//received response from server
						gameRooms[command[2]].gameRoomConn.SetReadDeadline(time.Time{})
						join_response := string(serverResponseBuffer[:nResponse])
						join_response_command := strings.Split(join_response, ":")
						if strings.Compare(join_response_command[0], "JOIN_SUCCESS") == 0 {
							var playerlist = "Join Success:"
							gameRooms[command[2]].players[command[1]] = clientConn
							gameRoomConn = gameRooms[command[2]].gameRoomConn
							//Successful Join, let's send response to client
							log.Printf("Client %s with IP %s has successfully joined the Game Room %s.\n", command[1], clientConn.RemoteAddr().String(), command[2])
							//create a string list of players that are part of the game room lobby right now
							for key := range gameRooms[command[2]].players {
								if strings.Compare(key, command[1]) != 0 {
									playerlist = playerlist + key + ","
								}
							}
							//send that list to the player
							err = clientConn.WriteMessage(1, []byte(playerlist[:len(playerlist)-1]))
							if err != nil {
								//player probably disconnected after joining, next iteration will handle disconnetion and comms to other players
								log.Printf("There was an issue while sending Join Success acknowledgment to the client with IP address : %s. Error : %s \n", clientConn.RemoteAddr(), err.Error())
								disconnect = true
								gameRoomMutex.Unlock()
								continue
							} else {
								//Send to the other clients that a user has joined!
								for key, value := range gameRooms[command[2]].players {
									//for every player in the game room hash, except the one that sent the request, inform them of the user join
									if strings.Compare(key, command[1]) != 0 {
										err = value.WriteMessage(1, []byte("User Join:"+command[1]))
										if err != nil {
											log.Printf("There was an issue while sending Join Success acknowledgment to player %s with IP address : %s. Error : %s \n", key, value.RemoteAddr(), err.Error())
											//We don't care, the client's own game routine needs to handle this.
										}
									}
								}
							}
						} else if strings.Compare(join_response_command[0], "Reconnect Success") == 0 {
							//update gameRoom connection for the client, to facilitate comms (it is the only way for the proxy to
							//know where to send the request at this specific point
							gameRoomConn = gameRooms[command[2]].gameRoomConn
							//re add the web socket of the player to the game room player list
							gameRooms[command[2]].players[command[1]] = clientConn
							log.Printf("Player %s has successfully reconnected to game with access code %s.\n", command[1], command[2])
							//Clean up later
							//var playerlist string = "";
							//send to the player who is currently playing the game. we will also send the currentRound & question. will test this later->we won't need this I don't think.
							err = clientConn.WriteMessage(1, []byte(join_response))
							if err != nil {
								log.Printf("There was an error with sending Reconnect Success to the client with IP : %s. Error : %s.\n", clientConn.RemoteAddr(), err.Error())
								//we need to handle disconnection again-> test
								keepServicing = false
							}
							//Don't think we need this
						} else if strings.Compare(join_response_command[0], "ROOM_FULL") == 0 {
							//Send to client that the room is full
							log.Printf("Player %s is unable to join Room %s. The room is currently full. Send to client Room Full.\n", command[1], command[2])
							err = clientConn.WriteMessage(1, []byte("Error:Room Full"))
							if err != nil {
								log.Printf("Room Full message was not sent to the client with IP address : %s. Error : %s \n", clientConn.RemoteAddr(), err.Error())
								//we don't care here if client disconnected, bye bye
								keepServicing = false
							}
						} else if strings.Compare(join_response_command[0], "USER_IN_ANOTHER_ROOM_ERROR") == 0 {
							//Send to client that the username is part of another room
							log.Printf("User trying to join is already in another room. Send to client with IP address : %s, Error:User in another room already.\n", clientConn.RemoteAddr())
							err = clientConn.WriteMessage(1, []byte("Error:User in another room already."))
							if err != nil {
								log.Printf("Error:User in another room already. was not sent to the client. Error : %s", err.Error())
								//we don't care about this. the username is in another room. if he disconnected bye bye
								keepServicing = false
							}
						} else if strings.Compare(join_response_command[0], "GAME_UNDERWAY") == 0 {
							//User tried to connect while a game is ongoing, send error message.
							err = clientConn.WriteMessage(1, []byte("Error:Game already underway. If you are trying to reconnect you need to use the same username as before."))
							if err != nil {
								log.Printf("Error:Game underway was not able to be transmitted to the user. Error : %s", err.Error())
								//we don't care about this. Game is underway. if he disconnected bye bye
								keepServicing = false
							}
						} else {
							//Unexpected response, should not happen
							log.Printf("Unexpected response from server for Join Room. Sending to client with IP address : %s, Error:Unexpected Join Room Server Response\n", clientConn.RemoteAddr())
							err = clientConn.WriteMessage(1, []byte("Error:Unexpected Join Room Server Response"))
							if err != nil {
								log.Printf("Error:Unexpected Join Room Server Response was not sent to the client. Error : %s\n", err.Error())
								//Assume that there was a server error, and the client didn't receive it, we don't care if client disconnected.
								keepServicing = false
							}
						}
					}
					service_completed = true
				} else {
					//Room doesn't exist, send error, this will likely happen if data gets corrupt, highly unlikely
					log.Printf("Room with access Code %s doesn't exists. Join Non Existent Room Error was sent to the client.\n", command[2])
					err = clientConn.WriteMessage(1, []byte("Error:Join Non Existent Room"))
					if err != nil {
						log.Printf("Join Non Existent Room Error was not sent to the client with IP addrees %s. Error : %s.\n", clientConn.RemoteAddr(), err.Error())
						//We don't care if he disconnects he is not participating in any room
						keepServicing = false
					}
				}
				gameRoomMutex.Unlock()
			} else if strings.Compare(command[0], "Answer") == 0 {
				//format should be of Answer:Username:AccessCode:Response Assume that the format of the request is correct.
				log.Printf("Answer Room Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				gameRoomMutex.Lock()
				log.Printf("Inside Answer Room Mutex.")
				//Check if game Room exists
				var _, ok = gameRooms[command[2]]
				if ok {
					//send command to server
					log.Printf("Sending Command Answer : %s", (string(buffer[:n]) + ":" + strconv.FormatInt(time_stamp, 10)))
                    gameRooms[command[2]].gameRoomConn.SetWriteDeadline(time.Now().Add(timeout))
					_, err = gameRooms[command[2]].gameRoomConn.Write([]byte(string(buffer[:n]) + ":" + strconv.FormatInt(time_stamp, 10)))
					if err != nil {
					    gameRooms[command[2]].gameRoomConn.SetWriteDeadline(time.Time{})
						log.Printf("Sending Answer Command %s to the server of game Room %s failed.\n", request, command[2])
						gameRoomConn = restartServer()
						gameRoomMutex.Unlock()
						service_completed = false
						continue
					} else {
						//Potential Race Condition, need timestamps
						gameRoomConn.SetWriteDeadline(time.Time{})
						gameRoomConn.SetReadDeadline(time.Now().Add(timeout))
						nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
						if err != nil {
							//failed to receive response from server, find new server
							log.Printf("Proxy was unable to receive a response from the server. Error : %s.\n", err.Error())
							gameRoomConn.SetReadDeadline(time.Time{})
							gameRoomConn = restartServer()
							gameRoomMutex.Unlock()
							service_completed = false
							continue
						} else {
						    gameRoomConn.SetReadDeadline(time.Time{})
							//Remember we need to start informing about disconnections
							//Successfully received response from server for answer question
							answer_response := string(serverResponseBuffer[:nResponse])
							log.Printf("Server Responded : %s.\n", answer_response)
							var answer_command = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
							if strings.Compare(answer_command[0], "Everyone Responded") == 0 {
								//First we need to handle the client that responded
								err = clientConn.WriteMessage(1, []byte(serverResponseBuffer[:nResponse]))
								if err != nil {
									//was unable to send message to client that sent the game request/
									//Will handle disconnection in next iteration of the for loop
									log.Printf("There was an issue while sending Everyone Responded acknowledgment to Player %s with IP address : %s. Error : %s. \n", command[1], clientConn.RemoteAddr(), err.Error())
									disconnect = true
								} else {
									//Successfully sent the message to the user that sent the request
									log.Printf("Sent Everyone Responded to player %s with IP address %s. This player made the last response\n", command[1], clientConn.RemoteAddr())
								}
								//Do we have more than 1 active players in the room? If yes we need to inform them that everyone answered
								//the question for them to see the next question of the room
								if len(gameRooms[command[2]].players) > 1 {
									//iterate through the list of active websocket connections
									for key, value := range gameRooms[command[2]].players {
										if strings.Compare(key, command[1]) != 0 {
											//if this player is not the one that is performing the request, send to them the information required to see the next question.
											err = value.WriteMessage(1, serverResponseBuffer[:nResponse])
											if err != nil {
												//Client that we are sending the everyone responded message disconnected, its own subroutine will handle this
												log.Printf("There was an issue while sending Everyone Responded acknowledgment to Player %s with IP address : %s. Error : %s. \n", key, value.RemoteAddr(), err.Error())
											} else {
												//Successful informing of player in the game Room.
												log.Printf("Sent Everyone Responded to player %s with IP address %s.\n", key, value.RemoteAddr())
											}
										}
									}
								}
							} else if strings.Compare(answer_command[0], "Game Over") == 0 {
								//Last Question was answered according to the server, Let's send this response to the player
								//that performed this request
								err = clientConn.WriteMessage(1, []byte(serverResponseBuffer[:nResponse]))
								if err != nil {
									//Game is over, it doesn't matter if the player disconnected
									log.Printf("There was an issue while sending Game Over message to the client with IP address : %s. Error : %s. \n", clientConn.RemoteAddr(), err.Error())
								} else {
									//Successfully sent the information to the player that made this request
									log.Printf("Sent Game Over to player %s.\n", command[1])
								}
								//If there is more than 1 active participant, we need to also inform them, that the game is over
								if len(gameRooms[command[2]].players) > 1 {
									for key, value := range gameRooms[command[2]].players {
										if strings.Compare(key, command[1]) != 0 {
											err = value.WriteMessage(1, []byte(serverResponseBuffer[:nResponse]))
											if err != nil {
												log.Printf("There was an issue while sending Game Over message to the client with IP address : %s. Error : %s. \n", value.RemoteAddr(), err.Error())
												//the client's subroutine will handle this we don't care!
											} else {
												log.Printf("Sent Game Over to player %s.\n", key)
											}
										}
									}
									//Need to check with Jason and see if we need to delete the individual players
								}
								log.Printf("The game is over for game Room %s, communications for this room will stop.\n", command[2])
								// Terminate all the websocket connections
								for _, value := range gameRooms[command[2]].players {
									value.Close()
								}
								// Terminating the tcp communications
								gameRooms[command[2]].gameRoomConn.Close()
								delete(gameRooms, command[2])
								gameRoomMutex.Unlock()
								break
							} else {
								//no action here. THere are no other types of answers that can be received by the server.
							}
						}
					}
					service_completed = true
				} else {
					//Invalid Accesscode provided -> potential data corruption
					log.Printf("Invalid Answer Access to the game.\n")
					keepServicing = false
				}
				gameRoomMutex.Unlock()
			} else if strings.Compare(command[0], "Ready") == 0 {
				//Setting Deadline to test
				//Ready to start request from client let's work Ready:Username:AccessCode
				log.Printf("Ready Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				gameRoomMutex.Lock()
				//we need to contact the game server, send the command
				log.Printf("Requesting from Server with address %s a Ready Request.\n", gameRooms[command[2]].gameRoomConn.RemoteAddr().String())
				// Checking for
				gameRooms[command[2]].gameRoomConn.SetWriteDeadline(time.Now().Add(timeout))
				_, err = gameRooms[command[2]].gameRoomConn.Write([]byte(string(buffer[:n]) + ":" + strconv.FormatInt(time_stamp, 10)))
				if err != nil {
					//Server Replication: Failure to send the command to the server, find another server
					log.Printf("Sending Ready Command %s to the server of game Room %s failed.\n", request, command[2])
					gameRooms[command[2]].gameRoomConn.SetWriteDeadline(time.Time{})
					gameRoomConn = restartServer()
					gameRoomMutex.Unlock()
					service_completed = false // For the current command to repeat
					continue
				}
				gameRooms[command[2]].gameRoomConn.SetWriteDeadline(time.Time{})
				gameRooms[command[2]].gameRoomConn.SetReadDeadline(time.Now().Add(timeout))
				nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
				if err != nil {
					log.Printf("Receiving Response from Server for Ready Failed. Send Server Response Error to client.")
					gameRooms[command[2]].gameRoomConn.SetReadDeadline(time.Time{})
					gameRoomConn = restartServer()
					gameRoomMutex.Unlock()
					service_completed = false // For the current command to repeat
					continue
				} else {
				    gameRooms[command[2]].gameRoomConn.SetReadDeadline(time.Time{})
					ready_response := string(serverResponseBuffer[:nResponse])
					var command = strings.Split(string(buffer[:n]), ":")
					var ready_command = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
					if strings.Compare(ready_command[0], "Ready Success") == 0 {
						//Successful READY, let's send response to client
						log.Printf("Client %s with IP %s has successfully set their status as ready to start the game.\n", command[1], clientConn.RemoteAddr().String())
						clientConn.WriteMessage(1, []byte(string(serverResponseBuffer[:nResponse])))
						if err != nil {
							log.Printf("There was an issue while sending Ready Success to Player %s with IP address : %s. Error : %s. \n", command[1], clientConn.RemoteAddr(), err.Error())
							disconnect = true
							gameRoomMutex.Unlock()
							continue
						} else {
							//successfully sent the Ready Success to client, let's inform other players
							for key, value := range gameRooms[command[2]].players {
								if strings.Compare(key, command[1]) != 0 {
									value.WriteMessage(1, []byte(string(serverResponseBuffer[:nResponse])))
									if err != nil {
										log.Printf("There was an issue while sending Ready Success to Player %s with IP address : %s. Error : %s. \n", key, value.RemoteAddr(), err.Error())
										//we don't care, the client that failed to receive the ready success message will be handled by its own go routine
									}
								}
							}
						}
					} else if strings.Compare(ready_command[0], "All Ready") == 0 {
						//Everyone is ready, let's send it to all players
						log.Printf("Client %s with IP %s has successfully set their status as ready to start the game and all players are ready to start.\n", command[1], clientConn.RemoteAddr().String())
						err = clientConn.WriteMessage(1, []byte(string(serverResponseBuffer[:nResponse])))
						//Weren't able to send response to player that initiated the request
						if err != nil {
							log.Printf("There was an issue while sending All Ready acknowledgment to player %s with IP address : %s. Error : %s. \n", command[1], clientConn.RemoteAddr(), err.Error())
							disconnect = true
							if len(gameRooms[command[2]].players) > 1 {
								//if there are more players it must be the case that they are all ready to play!
								for key, value := range gameRooms[command[2]].players {
									err = value.WriteMessage(1, []byte(string(serverResponseBuffer[:nResponse])))
									if err != nil {
										log.Printf("There was an issue while sending Ready Success All Players Ready Success acknowledgment to the client with IP address : %s. Error : %s. \n", clientConn.RemoteAddr(), err.Error())
										//one of the players is disconnected, this will be handled by the players own go routine.
									} else {
										log.Printf("Sent All Ready to player %s.\n", key)
									}
								}
							} else {
								//don't think we need check back later
								//we need to shut the gameRoom Connection once we figure out the server logic
							}
						} else {
							//We were able to send request to player that initiated the request
							for key, value := range gameRooms[command[2]].players {
								//Send the response to all players except for the one that initiated the request
								if strings.Compare(key, command[1]) != 0 {
									err = value.WriteMessage(1, []byte(string(serverResponseBuffer[:nResponse])))
									if err != nil {
										log.Printf("There was an issue while sending Ready Success All Players Ready Success acknowledgment to the client with IP address : %s. Error : %s. \n", clientConn.RemoteAddr(), err.Error())
										//one of the players is disconnected, this will be handled by the players own go routine.
									} else {
										log.Printf("Sent All Ready to player %s.\n", key)
									}
								}
							}
						}
					} else {
						//doesn't matter won't occur. Test later
						log.Printf("The server sent the following response : %s\n", ready_response)
						err = clientConn.WriteMessage(1, []byte("Error:"+ready_response))
						if err != nil {
							log.Printf("There was an issue while sending the servers' answer to the client. \n")
							//probably just close connection? -> this will be checked after the full implementation
						}
					}
				}
				service_completed = true
				gameRoomMutex.Unlock()
			} else {
				//This is unlikely to happen. We should do some more testing, the trivia browser won't send a request that we don't have available
				log.Printf("Request %s has an in correct format. COMMUNICATION_PROTOCOL_ERROR will be sent to the client.\n", string(buffer[:n]))
				err = clientConn.WriteMessage(1, []byte("Error:Invalid Command Sent"))
				if err != nil {
					log.Printf("Sending COMMUNICATION_PROTOCOL_ERROR to client with IP address failed %s. Error : %s.", clientConn.RemoteAddr().String(), err.Error())
					keepServicing = false
				}
			}
			service_completed = true
		} else {
			//Cannot Service request, authentication information missing, this is also unlikely to happen-> do some testing! - Probably if we have an illegal attempt
			//of connection, i.e., from a non trivia browser.
			log.Printf("Communications Protocol Violated. Error will be sent to client, and connection terminated.\n")
			log.Printf("Command : %s\n", string(buffer[:n]))
			err = clientConn.WriteMessage(1, ([]byte("Error:Invalid Communication Protocol.")))
			if err != nil {
				log.Printf("Sending COMMUNICATION_PROTOCOL_ERROR to client with IP address %s failed. Error %s.", clientConn.RemoteAddr().String(), err.Error())
				keepServicing = false
			}
		}
	}
	clientConn.Close()
}
