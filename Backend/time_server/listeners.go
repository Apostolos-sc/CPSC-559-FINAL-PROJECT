package main

import (
	"log"
	"net/http"
)

func serverListener() {
	http.HandleFunc("/ws/server", wsEndpointServer)
    log.Printf("Attempting to Listen on %v:%v for server Requests.\n", SERVER_REGISTRATION.host, SERVER_REGISTRATION.port)
	err := http.ListenAndServe(SERVER_REGISTRATION.host+":"+SERVER_REGISTRATION.port, nil) // Is this line a loop??
	if err != nil {
		log.Printf("Unable to Listen and Serve HTTP Requests on %s:%s. Error : %s.\n", SERVER_REGISTRATION.host, SERVER_REGISTRATION.port, err.Error())
		//Need to handle error. Potential kill process.
	}
}

func clientListener() {
	log.Printf("In client listener")
	http.HandleFunc("/ws/client", wsEndpointClient)
    log.Printf("Attempting to listen on %v:%v for Client Requests.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port)
	err := http.ListenAndServe(CLIENT_SERVICE.host+":"+CLIENT_SERVICE.port, nil) // Is this line a loop??
	if err != nil {
		log.Printf("Unable to Listen and Serve HTTP Requests on %s:%s. Error : %s.\n", CLIENT_SERVICE.host, CLIENT_SERVICE.port, err.Error())
		//Need to handle error. Potential kill process.
	}

}

var clientCounter = 0

func wsEndpointClient(w http.ResponseWriter, r *http.Request) {
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

func wsEndpointServer(w http.ResponseWriter, r *http.Request) {
	log.Printf("In ws endpoint")
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	// upgrade this connection to a WebSocket Connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("There was an error when attempting to upgrade connection to a web socket. Error : %s\n", err.Error())
	}
	handleServerRegistration(ws)
}