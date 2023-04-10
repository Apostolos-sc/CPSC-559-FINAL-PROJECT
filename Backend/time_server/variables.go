package main

import (
	"github.com/gorilla/websocket"
	"sync"
)

type connection struct {
	host     string
	port     string
	con_type string
}

var SERVER_REGISTRATION_1 = connection{"10.0.0.105", "6609", "tcp"}
var SERVER_REGISTRATION_2 = connection{"10.0.0.105", "6610", "tcp"}
var err error
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	gameRoomMutex sync.Mutex
	gameRooms     = make(map[string]map[string]*websocket.Conn) // accessCode: { username : { websocket conn}}
)
