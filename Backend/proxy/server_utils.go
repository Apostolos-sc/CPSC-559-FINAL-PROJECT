package main

import (
	"log"
	"net"
	"strings"
	"time"
)

func disconnectServerInform(roomConnection net.Conn, username string, accessCode string) {
	log.Printf("User %s, is trying to disconnect from room %s", username, accessCode)
	_, err := roomConnection.Write([]byte("Disconnect:" + username + ":" + accessCode))
	if err != nil {
		log.Printf("Sending Disconnect information to the server Discconect:%s:%s to server %s failed. Error : %s\n", roomConnection.RemoteAddr().String(), err.Error())
		//handle error -> server crashed, need to switch servers
	}
}

func disconnectClientInforming(accessCode string, username string) {
	//for key, value in range:
}

func handleServerRegistration(conn net.Conn) {
	//Server Registration Handler
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("There was an issue with reading from the potential Server with IP %s. Error : %s.\n", conn.RemoteAddr(), err.Error())
	}
	if strings.Compare(string(buffer[:n]), "Server Join") == 0 {
		//Client attempting to connect is a server
		host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			log.Println("There was an error while splitting the remote Address of server. Error : %s.\n", err.Error())
		}
		//lock the servers_slice variable

		time := time.Now().Format(time.ANSIC)
		log.Printf("Command : %v. Send Accepted.\n", string(buffer[:]))
		conn.Write([]byte("Accepted"))
		n, err = conn.Read(buffer)
		if err != nil {
			log.Printf("There was an error while waiting to received the IP address on which the server with IP address : %s, will be servicing game rooms. Error : %s.\n", conn.RemoteAddr(), err.Error())
		}
		//We assume that the server next will send the IP address for which he will be listening on.
		host, port, err = net.SplitHostPort(string(buffer[:n]))
		if err != nil {
			log.Printf("There was an error while waiting extracting the IP address on which the server with IP address : %s, will be servicing game rooms. Error : %s.\n", conn.RemoteAddr(), err.Error())
		} else {
			log.Printf("Successfully extracted the address on which the server with IP address : %s, will be servicing game rooms on : %s:%s.\n", conn.RemoteAddr(), host, port)
			serverMutex.Lock()
			//Add server data on the server slice
			serverList = append(serverList, connection{host: host, port: port, con_type: "tcp"})
			//set total number of games serving to zero
			serverMutex.Unlock()
			log.Printf("%s was added as a server on the server list on %v.\n", string(buffer[:n]), time)
			//send back that address was received to let know the server that all is OKAY
			_, err = conn.Write([]byte("Received Address"))
			if err != nil {
				log.Printf("There was an issue while sending Address Acknowledgement to the server with IP address : %s. Error : %s.\n", conn.RemoteAddr(), err.Error())
			}
		}
	} else {
		_, err = conn.Write([]byte("Wrong command given, access declined."))
		if err != nil {
			log.Printf("There was an issue while informing the potential server that the given command is incorrect. \n %s", err.Error())
		}
	}
	// close conn
	conn.Close()
}
