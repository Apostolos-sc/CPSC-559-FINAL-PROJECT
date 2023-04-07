package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net"
	"strings"
)

func handleClientRequest(clientConn *websocket.Conn, connID int) {
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
						disconnectServerInform(gameRoomConn, previousCommandString[1], previousCommandString[2])
						delete(gameRooms[previousCommandString[2]].players, previousCommandString[1])
						gameRoomConn.Close()
						delete(gameRooms, previousCommandString[2])
						gameRoomMutex.Unlock()
						log.Printf("Game Room : %s has stopped servicing players.\n", previousCommandString[2])
						break
					} else {
						log.Printf("There are more players in the room{%d}, let's contact the server, delete the user: %s\n", len(gameRooms[previousCommandString[2]].players), previousCommandString[1])
						// The game room can be null when the previousCommand is not CreateGameRoom but JoinRoom
						// Test this, added on line 397
						if gameRoomConn == nil {
							gameRoomConn = gameRooms[previousCommandString[2]].gameRoomConn
						}
						disconnectServerInform(gameRoomConn, previousCommandString[1], previousCommandString[2])
						delete(gameRooms[previousCommandString[2]].players, previousCommandString[1])
						nResponse, err = gameRooms[previousCommandString[2]].gameRoomConn.Read(serverResponseBuffer)
						if err != nil {
							log.Printf("There was an error while waiting for the server to respond on disconnect command. Error : %s\n", err.Error())
						} else {
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
										log.Printf("There was an issue while sending Everyone Responded acknowledgment to Player %s with IP address : %s. Error : %s. \n", key, value.RemoteAddr() ,err.Error())
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
										log.Printf("There was an issue while sending Everyone Responded acknowledgment to Player %s with IP address : %s. Error : %s. \n", key, value.RemoteAddr() ,err.Error())
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
		log.Printf("Waiting to read from client with ID %d.\n", connID)
		_, buffer, err = clientConn.ReadMessage()
		if err != nil {
			log.Println("Failed to read a request from the client. Connection will be terminated.")
			log.Printf("Previous command : %s.\n", previousCommand)
			disconnect = true
			continue
		} else {
			log.Printf("Message received from client with ID : %d.\n", connID)
		}
		//Assign this command as the one that was executed before
		n = len(buffer)
		previousCommand = string(buffer[:len(buffer)])
		log.Printf("Previous command set as :%s", previousCommand[:n])
		//critical access
		if len(serverList) == 0 {
			//Can't service Client, no live Servers.
			log.Println("There are no servers available to service Clients. Send Error to Client. Connection Terminated.")
			err = clientConn.WriteMessage(1, ([]byte("Error:No Server Available")))
			if err != nil {
				log.Println("Unable to send to client Error: No Server Available:", err.Error())
			}
			keepServicing = false
			continue
		}
		var command = strings.Split(string(buffer[:n]), ":")
		var request = string(buffer[:n])
		//check if proxy received a valid request
		if checkRequest(command) {
			if strings.Compare(command[0], "Create Room") == 0 {
				//Critical Section
				gameRoomMutex.Lock()
				gameRoomAddr, err := net.ResolveTCPAddr("tcp", serverList[0].host+":"+serverList[0].port)
				address := serverList[0].host + ":" + serverList[0].port
				if err != nil {
					log.Printf("ResolveTCPAddr failed for %s:%v\n", address, err.Error())
					//Need to handle the case where we can't resolve the server.
				}
				log.Printf("Create Room Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				//attempt to connect to server to establish a game room connection using a tcp
				gameRoomConn, err = net.DialTCP("tcp", nil, gameRoomAddr)
				if err != nil {
					log.Printf("Dialing server %s for game Room Creation Request failed: %v\n", address, err.Error())
					//Need to handle the case where we can't resolve the server
				}
				//forward gameRoom creation request to server
				log.Printf("Requesting from Server with address %s Game Room Creation.\n", gameRoomConn.RemoteAddr().String())
				_, err = gameRoomConn.Write(buffer[:n])
				if err != nil {
					log.Printf("Sending the Game Room Creation Command : %s to the server failed: %s", string(buffer[:n]), err.Error())
					//Server Crashed, time to find a new server
				}
				//wait for response
				nResponse, err = gameRoomConn.Read(serverResponseBuffer)
				if err != nil {
					log.Println("Read data failed:", err.Error())
					log.Printf("Failed to receive a response from server : %s. Will attempt to connect to another server.\n", gameRoomConn.RemoteAddr().String())
					//failed to read from server, let's try to connect to another server and retry
					//will fix soon
					gameRoomConn.Close()
					continue
				}
				var response = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
				if strings.Compare(response[0], "Room Created") == 0 {
					//acknowledgment received. Room was received
					//serviceComplete = true;
					log.Printf("Game Room successfully created. Access Code : %s and is served by : %s\n", response[1], gameRoomConn.RemoteAddr().String())
					//Critical Section Lock
					//Initialiaze game room struct and assign to map: gameRooms[accessCode] = gameRoom
					gameRooms[response[1]] = gameRoom{gameRoomConn, make(map[string]*websocket.Conn)}
					//player should be added to the player hashmap in the gameRoom with accessCode : response[1]
					gameRooms[response[1]].players[command[1]] = clientConn
					//Send acknowledgement to server that proxy received access code
					_, err = gameRoomConn.Write([]byte("Access Code Received"))
					if err != nil {
						log.Printf("Sending Acknowledgement that the code was received to server %s failed : %s\n", gameRoomConn.RemoteAddr().String(), err.Error())
						//handle error -> server crashed, time to find a new server
					}
					//Send to client success message -> If can't write, assume client disconnection, else print log info
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
				gameRoomMutex.Unlock()
			} else if strings.Compare(command[0], "Join Room") == 0 {
				//First let's check if the room exists - Lock Critical Resource first - We need this to redirect the traffic
				log.Printf("Join Room Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				gameRoomMutex.Lock()
				_, ok := gameRooms[command[2]]
				//The Game Room exists
				if ok == true {
					log.Printf("Requesting from Server with address %s a Join Room Request.\n", gameRooms[command[2]].gameRoomConn.RemoteAddr().String())
					_, err = gameRooms[command[2]].gameRoomConn.Write(buffer[:n])
					if err != nil {
						log.Printf("Sending Join Room Command %s to the server of game Room %s failed.\n", request, command[2])
						//Server disconnected -> we need to handle this
					}
					nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
					if err != nil {
						log.Printf("Receiving Response from Server for Join Room Failed. SEND SERVER_RESPONSE_ERROR to client\n")
						//we need to fix this. Server crashed-
						err = clientConn.WriteMessage(1, []byte("Error:Server Response"))
						if err != nil {
							log.Printf("SERVER_RESPONSE_ERROR was not sent to the client. Error : %s", err.Error())
							//we don't care if the client disconnects at this point.
							keepServicing = false
						}
					} else {
						//received response from server
						join_response := string(serverResponseBuffer[:nResponse])
						join_response_command := strings.Split(join_response, ":")
						if strings.Compare(join_response_command[0], "JOIN_SUCCESS") == 0 {
							var playerlist = "Join Success:"
							gameRooms[command[2]].players[command[1]] = clientConn
							gameRoomConn = gameRooms[command[2]].gameRoomConn
							//Successful Join, let's send response to client
							log.Printf("Client %s with IP %s has successfully joined the Game Room %s.\n", command[1], clientConn.RemoteAddr().String(), command[2])
							for key := range gameRooms[command[2]].players {
								if strings.Compare(key, command[1]) != 0 {
									playerlist = playerlist + key + ","
								}
							}
							err = clientConn.WriteMessage(1, []byte(playerlist[:len(playerlist)-1]))
							if err != nil {
								log.Printf("There was an issue while sending Join Success acknowledgment to the client with IP address : %s. Error : %s \n", clientConn.RemoteAddr(), err.Error())
								disconnect = true
								gameRoomMutex.Unlock()
								continue
							} else {
								//Send to the other clients that a user has joined!
								for key, value := range gameRooms[command[2]].players {
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
							gameRoomConn = gameRooms[command[2]].gameRoomConn
							gameRooms[command[2]].players[command[1]] = clientConn
							log.Printf("Player %s has successfully reconnected to game with access code %s.\n", command[1], command[2])
							//var playerlist string = "";
							//send to the player who is currently playing the game. we will also send the currentRound & question. will test this later->we won't need this I don't think.
							err = clientConn.WriteMessage(1, []byte(join_response))
							if err != nil {
								log.Printf("There was an error with sending Reconnect Success to the client with IP : %s. Error : %s.\n", clientConn.RemoteAddr(), err.Error())
								keepServicing = false
							}
							//Don't think we need this
						} else if strings.Compare(join_response_command[0], "ROOM_FULL") == 0 {
							log.Printf("Player %s is unable to join Room %s. The room is currently full. Send to client Room Full.\n", command[1], command[2])
							err = clientConn.WriteMessage(1, []byte("Error:Room Full"))
							if err != nil {
								log.Printf("Room Full message was not sent to the client with IP address : %s. Error : %s \n", clientConn.RemoteAddr(), err.Error())
								//we don't care here if client disconnected, bye bye
								keepServicing = false
							}

						} else if strings.Compare(join_response_command[0], "USER_IN_ANOTHER_ROOM_ERROR") == 0 {
							log.Printf("User trying to join is already in another room. Send to client with IP address : %s, Error:User in another room already.\n", clientConn.RemoteAddr())
							err = clientConn.WriteMessage(1, []byte("Error:User in another room already."))
							if err != nil {
								log.Printf("Error:User in another room already. was not sent to the client. Error : \n", err.Error())
								//we don't care about this. the username is in another room. if he disconnected bye bye
								keepServicing = false
							}
						} else if strings.Compare(join_response_command[0], "GAME_UNDERWAY") == 0 {
							err = clientConn.WriteMessage(1, []byte("Error:Game already underway. If you are trying to reconnect you need to use the same username as before."))
							if err != nil {
								log.Printf("Error:Game underway was not able to be transmitted to the user. Error : \n", err.Error())
								//we don't care about this. Game is underway. if he disconnected bye bye
								keepServicing = false
							}
						} else {
							log.Printf("Unexpected response from server for Join Room. Sending to client with IP address : %s, Error:Unexpected Join Room Server Response\n", clientConn.RemoteAddr())
							err = clientConn.WriteMessage(1, []byte("Error:Unexpected Join Room Server Response"))
							if err != nil {
								log.Printf("Error:Unexpected Join Room Server Response was not sent to the client. Error : \n", err.Error())
								//Assume that there was a server error, and the client didn't receive it, we don't care if client disconnected.
								keepServicing = false
							}
						}
					}
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
				var _, ok = gameRooms[command[2]]
				if ok {
					_, err = gameRooms[command[2]].gameRoomConn.Write(buffer[:n])
					if err != nil {
						log.Printf("Sending Answer Command %s to the server of game Room %s failed.\n", request, command[2])
						//we need to do server replication here
					} else {
						nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
						if err != nil {
							log.Printf("Proxy was unable to receive a response from the server. Error : %s.\n", err.Error())
							//Replication
						} else {
							//Remember we need to start informing about disconnections
							answer_response := string(serverResponseBuffer[:nResponse])
							log.Printf("Server Responded : %s.\n", answer_response)
							var answer_command = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
							if strings.Compare(answer_command[0], "Everyone Responded") == 0 {
								//First we need to handle the client that responded
								err = clientConn.WriteMessage(1, []byte(serverResponseBuffer[:nResponse]))
								if err != nil {
									log.Printf("There was an issue while sending Everyone Responded acknowledgment to Player %s with IP address : %s. Error : %s. \n", clientConn.RemoteAddr(), err.Error())
									disconnect = true
								} else {
									log.Printf("Sent Everyone Responded to player %s with IP address %s. This player made the last response\n", command[1], clientConn.RemoteAddr())
								}
								//we need to check and see if this is the last player that was playing and if he disconnected
								if len(gameRooms[command[2]].players) > 1 {
									for key, value := range gameRooms[command[2]].players {
										if strings.Compare(key, command[1]) != 0 {
											err = value.WriteMessage(1, []byte(serverResponseBuffer[:nResponse]))
											if err != nil {
												log.Printf("There was an issue while sending Everyone Responded acknowledgment to Player %s with IP address : %s. Error : %s. \n", key, err.Error())
												//Client that we are sending the everyone responded message disconnected, its own subroutine will handle this
											} else {
												log.Printf("Sent Everyone Responded to player %s with IP address %s.\n", key, value.RemoteAddr())
											}
										}
									}
								}
								//we check to see if there are other players in the room. This ensures if there was a disconnection and it wasn't the last
								//disconnected in the room the message is sent to everyone
							} else if strings.Compare(answer_command[0], "Game Over") == 0 {
								//Let's inform the person that made this request
								err = clientConn.WriteMessage(1, []byte(serverResponseBuffer[:nResponse]))
								if err != nil {
									log.Printf("There was an issue while sending Game Over message to the client with IP address : %s. Error : %s. \n", clientConn.RemoteAddr(), err.Error())
									//Game is over, it doesn't matter if the player disconnected
									//bye bye
								} else {
									log.Printf("Sent Game Over to player %s.\n", command[1])
								}
								// Send game over request to all other participants
								if len(gameRooms[command[2]].players) > 0 {
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
								for _, value := range gameRooms[command[2]].players {
									value.Close()
								}
								gameRooms[command[2]].gameRoomConn.Close()
								delete(gameRooms, command[2])
								gameRoomMutex.Unlock()
								break
							} else {
								//no action here. we could use for leaderboard information during the game if needed
							}
						}
					}
				} else {
					log.Printf("Invalid Answer Access to the game.\n")
					keepServicing = false
				}
				gameRoomMutex.Unlock()
			} else if strings.Compare(command[0], "Ready") == 0 {
				//Ready to start request from client let's work Ready:Username:AccessCode
				log.Printf("Ready Request : %s. Client's IP : %s\n", request, clientConn.RemoteAddr())
				gameRoomMutex.Lock()
				//we need to contact the game server, send the command
				log.Printf("Requesting from Server with address %s a Ready Request.\n", gameRooms[command[2]].gameRoomConn.RemoteAddr().String())
				_, err = gameRooms[command[2]].gameRoomConn.Write(buffer[:n])
				if err != nil {
					log.Printf("Sending Ready Command %s to the server of game Room %s failed.\n", request, command[2])
					//Server Replication handling
				}
				nResponse, err = gameRooms[command[2]].gameRoomConn.Read(serverResponseBuffer)
				if err != nil {
					//Server replication to be handled next
					log.Printf("Receiving Response from Server for Ready Failed. Send Server Response Error to client.\n")
					err = clientConn.WriteMessage(1, []byte("Error:Server Failed to Respond"))
					if err != nil {
						log.Printf("Server Response Error was not sent to the client with IP Address %s. Error : %s.\n", clientConn.RemoteAddr(), err.Error())
						//handle, client disconnected?
					}
				} else {
					ready_response := string(serverResponseBuffer[:nResponse])
					var command = strings.Split(string(buffer[:n]), ":")
					var ready_command = strings.Split(string(serverResponseBuffer[:nResponse]), ":")
					if strings.Compare(ready_command[0], "Ready Success") == 0 {
						//Successful READY, let's send response to client
						log.Printf("Client %s with IP %s has successfully set their status as ready to start the game.\n", command[1], clientConn.RemoteAddr().String())
						clientConn.WriteMessage(1, []byte(string(serverResponseBuffer[:nResponse])))
						if err != nil {
							log.Printf("There was an issue while sending Ready Success to Player %s with IP address : %s. Error : %s. \n", command[1], clientConn.RemoteAddr(), err.Error())
							/*
								disconnectServerInform(gameRooms[command[2]].gameRoomConn, command[1], command[2])
								//we might have to close the connection here!! -> test
								//remove player from local hashmap of the gameRoom connection
								delete(gameRooms[command[2]].players, command[1])
								keepServicing = false */
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
							/*
								disconnectServerInform(gameRooms[command[2]].gameRoomConn, command[1], command[2])
								//we might have to close the connection here!! -> test
								//remove player from local hashmap of the gameRoom connection
								delete(gameRooms[command[2]].players, command[1])
								keepServicing = false */
							disconnect = true
							if len(gameRooms[command[2]].players) > 0 {
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
						log.Printf("The server sent the following response : %s\n", ready_response)
						err = clientConn.WriteMessage(1, []byte("Error:"+ready_response))
						if err != nil {
							log.Printf("There was an issue while sending the servers' answer to the client. \n")
							//probably just close connection? -> this will be checked after the full implementation
						}
					}
				}
				gameRoomMutex.Unlock()
			} else {
				//This is unlikely to happen. We should do some more testing, the trivia browser won't send a request that we don't have available
				log.Printf("Request %s has an in correct format. COMMUNICATION_PROTOCOL_ERROR will be sent to the client.\n")
				err = clientConn.WriteMessage(1, []byte("Error:Invalid Command Sent"))
				if err != nil {
					log.Println("Sending COMMUNICATION_PROTOCOL_ERROR to client with IP address failed %s. Error : %s.", clientConn.RemoteAddr().String(), err.Error())
					keepServicing = false
				}
			}
		} else {
			//Cannot Service request, authentication information missing, this is also unlikely to happen-> do some testing! - Probably if we have an illegal attempt
			//of connection, i.e., from a non trivia browser.
			log.Printf("Communications Protocol Violated. Error will be sent to client, and connection terminated.\n")
			log.Printf("Command : %s\n", string(buffer[:n]))
			err = clientConn.WriteMessage(1, ([]byte("Error:Invalid Communication Protocol.")))
			if err != nil {
				log.Println("Sending COMMUNICATION_PROTOCOL_ERROR to client with IP address %s failed. Error %s.", clientConn.RemoteAddr().String(), err.Error())
				keepServicing = false
			}
		}
	}
	clientConn.Close()
}
