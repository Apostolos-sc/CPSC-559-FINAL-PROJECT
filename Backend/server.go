//Author         : Apostolos Scondrianis
//Created On     : 28-02-2023
//Last Edited By : Apostolos Scondrianis
//Last Edit On   : 01-03-2023
//Filename       : server.go
//Version        : 0.2

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type connection struct {
	host string;
	port string;
	con_type string;
}
var PROXY = connection{"127.0.0.1", "9000", "tcp"}
var GAME_SERVICE = connection{"127.0.0.1", "8080", "tcp"}

func main() {
	if(connectToProxy()) {
		//Listen for Game Service Requests if you were successfully registered at the Proxy
		fmt.Println("Successful registration to proxy.")
		gameServiceTCPAddr, err := net.ResolveTCPAddr(GAME_SERVICE.con_type, GAME_SERVICE.host+":"+GAME_SERVICE.port)
		if err != nil {
			fmt.Printf("Unable to resolve IP")
		}
	
		// Start TCP Listener for the server
		listener, err := net.ListenTCP("tcp", gameServiceTCPAddr)
		if err != nil {
			fmt.Printf("Unable to start listener - %s", err)
		} else {
			fmt.Printf("Listening on %v:%v\n", GAME_SERVICE.host, GAME_SERVICE.port)
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
			fmt.Printf("Incoming connection from : %s\n", conn.RemoteAddr().String())
			go handleGameRequest(conn)
		}
	} else {
		fmt.Println("Failed to register server at the proxy.")
	}
}

func handleGameRequest(conn net.Conn) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	if strings.Compare(string(buffer[:n]), "Game Room") == 0 {
		//Game Connection Attempt.
		conn.Write([]byte("Game Room ID"))
		n, err = conn.Read(buffer)
		if err != nil {
			log.Fatal(err)
		}
		roomID, err := strconv.Atoi(string(buffer[:n]))
		if err != nil {
			fmt.Println("Error during conversion")
		} else {
			fmt.Printf("Proxy wants us to handle Game Room with ID %d\n", roomID)
		}
	} else {
		fmt.Println("Invalid Option Given by the proxy.")
	}
	// close conn
	conn.Close()
}

//Returns true if the proxy accepts the connection.
func connectToProxy() bool {
	proxyAddr, err := net.ResolveTCPAddr(PROXY.con_type, PROXY.host+":"+PROXY.port)
	if err != nil {
		fmt.Println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}
	//attempt to connect to proxy using a tcp connection
	conn, err := net.DialTCP(PROXY.con_type, nil, proxyAddr)
	if err != nil {
		fmt.Println("Dial failed:", err.Error())
		os.Exit(1)
	}

	_, err = conn.Write([]byte("Server Join"))
	if err != nil {
		fmt.Println("Write data failed:", err.Error())
		os.Exit(1)
	}

	// buffer to get data
	received := make([]byte, 8192)
	n, err := conn.Read(received)
	if err != nil {
		fmt.Println("Read data failed:", err.Error())
		os.Exit(1)
	}
	if strings.Compare(string(received[:n]), "Accepted") == 0 {
		//If the proxy accepted us, send the address we will be serving at
		fmt.Printf("Received message: %s.\n", string(received[:n]))
		var gameServiceAddress = GAME_SERVICE.host+":"+GAME_SERVICE.port
		_, err = conn.Write([]byte(gameServiceAddress))
		if(err != nil) {
			fmt.Println("Write data failed:", err.Error())
			os.Exit(1)
		} else {
			//wait for acknowledgement.
			n, err = conn.Read(received)
			if err != nil {
				fmt.Println("Read data failed:", err.Error())
				os.Exit(1)
			} else {
				//proxy received address success.
				if strings.Compare(string(received[:n]), "Received Address") == 0 {
					fmt.Printf("Received message: %s.\n", string(received[:n]))
					conn.Close()
					return true;
				}
			}
		}
	}
	conn.Close()
	return false;
}