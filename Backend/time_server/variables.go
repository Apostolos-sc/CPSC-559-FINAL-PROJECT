package main

import (
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

type connection struct {
	host     string
	port     string
	con_type string
}

var SERVER_REGISTRATION_1 = connection{"10.0.0.2", "6609", "tcp"}

// var SERVER_REGISTRATION_2 = connection{"10.0.0.8", "6610", "tcp"}
var err error
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type gameRoom struct {
	currentRound int
    ticker       *time.Ticker
    localMutex   sync.RWMutex
	players      map[string]*websocket.Conn
}

var (
	gameRoomMutex sync.Mutex
	gameRooms     = make(map[string]*gameRoom) // accessCode: { username : { websocket conn}}
)

var CLIENT_SERVICE = connection{"10.0.0.2", "7000", "tcp"}

var allowed_time int = 35
