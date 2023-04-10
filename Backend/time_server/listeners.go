package main

import (
	"log"
	"net"
	"net/http"
	"os"
)

func serverListener() {

	log.Printf("In server Listener")
	serverRegistrationTCPAddr, err := net.ResolveTCPAddr(SERVER_REGISTRATION_1.con_type, SERVER_REGISTRATION_1.host+":"+SERVER_REGISTRATION_1.port)
	if err != nil {
		log.Printf("Unable to resolve IP address for server registration on the time server.\n")
	}

	// Start TCP Listener
	listener, err := net.ListenTCP("tcp", serverRegistrationTCPAddr)
	if err != nil {
		log.Printf("Unable to start the time_server listener - %s", err.Error())
	} else {
		log.Printf("Listening on %v:%v for Server Registration Requests.\n", SERVER_REGISTRATION_1.host, SERVER_REGISTRATION_1.port)
	}
	//close Listener
	defer listener.Close()

	// Continuously Listen for connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Server Listener Accept functionality error occurred:", err.Error())
			os.Exit(1)
		}
		log.Printf("Potential Server Registration Request Incoming from : %s\n", conn.RemoteAddr().String())
		go handleServerRegistration(conn)
	}
}

func clientListener() {
	log.Printf("In client listener")
	http.HandleFunc("/ws", wsEndpoint)
}

var clientCounter = 0

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Printf("In ws endpoint")
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	// upgrade this connection to a WebSocket Connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("There was an error when attempting to upgrade connection to a web socket. Error : %s\n", err.Error())
	}
	clientCounter++
	handleClientRequest(ws, clientCounter)
}
