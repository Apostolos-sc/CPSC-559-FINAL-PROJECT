package main

import (
	"log"
	"net"
	"net/http"
	"os"
)

func clientListener() {
	//Client needs to provide which game room ID it is going be to connecting to.
	//websocket handler - no error handling needed here
	http.HandleFunc("/ws", wsEndpoint)
	//http listener
	log.Printf("Listening on %v:%v for Client Requests.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port)
	err := http.ListenAndServe(CLIENT_SERVICE.host+":"+CLIENT_SERVICE.port, nil)
	if err != nil {
		log.Printf("Unable to Listen and Serve HTTP Requests on %s:%s. Error : %s.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port, err.Error())
		//Need to handle error. Potential kill process.
	}
	log.Printf("Listening on %v:%v for client Requests.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port)

}

func serverListener() {
	// Resolve TCP Address
	//Address to be listening on
	serverRegistrationTCPAddr, err := net.ResolveTCPAddr(SERVER_REGISTRATION.con_type, SERVER_REGISTRATION.host+":"+SERVER_REGISTRATION.port)
	if err != nil {
		log.Printf("Unable to resolve IP address for server registration on the proxy server.\n")
	}

	// Start TCP Listener
	listener, err := net.ListenTCP("tcp", serverRegistrationTCPAddr)
	if err != nil {
		log.Printf("Unable to start the proxy listener listener - %s", err.Error())
	} else {
		log.Printf("Listening on %v:%v for Server Registration Requests.\n", SERVER_REGISTRATION.host, SERVER_REGISTRATION.port)
	}
	//close Listener
	defer listener.Close()

	//Continuously Listen for connections
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
