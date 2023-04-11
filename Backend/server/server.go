// Hashmap may be an issue? Maybe we want multiple servers to handle
// the same gameRoom incase one server is overly busy, think about
// during fault tolerance, replication stage, scalability

package main

import (
    "github.com/gorilla/websocket"
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"net/url"
)

type Question struct {
	ID       int
	question string
	answer   string
	option_1 string
	option_2 string
	option_3 string
	option_4 string
}

// ready should be 0 or 1 to indicate if a player is ready to start
type roomUser struct {
	username               string
	accessCode             string
	points                 int
	ready                  int
	offline                int
	roundAnswer            int
	correctAnswer          int
	accessCodeTimeStamp    int64
	pointsTimeStamp        int64
	readyTimeStamp         int64
	offlineTimeStamp       int64
	roundAnswerTimeStamp   int64
	correctAnswerTimeStamp int64
}
type connection struct {
	host     string
	port     string
	con_type string
}

type gameRoom struct {
	accessCode                           string
	currentRound                         int
	numOfPlayersAnswered                 int
	numOfPlayersAnsweredCorrect          int
	numOfDisconnectedPlayers             int
	accessCodeTimeStamp                  int64
	currentRoundTimeStamp                int64
	numOfPlayersAnsweredTimeStamp        int64
	numOfPlayersAnsweredCorrectTimeStamp int64
	numOfDisconnectedPlayersTimeStamp    int64
	questions                            map[int]*Question
	players                              map[string]*roomUser
}

var (
	gameRoomsMutex sync.Mutex
	gameRooms      = make(map[string]*gameRoom)
)

var proxy_ip_address = "10.0.0.2"
var MAX_PLAYERS = 4
var MAX_ROUNDS = 10
var PROXY = connection{proxy_ip_address, "9000", "tcp"}
var GAME_SERVICE = connection{proxy_ip_address, "8082", "tcp"}
var SERVER_LISTENER = connection{proxy_ip_address, "7000", "tcp"}
var nginx_master = connection{"10.0.0.2", "4406", "tcp"}
var nginx_slave = connection{"10.0.0.2", "5506", "tcp"}
var TIME_SERVER_1 = connection{"10.0.0.2", "6609", "tcp"}
var TIME_SERVER_2 = connection{"10.0.0.2", "6610", "tcp"}
var db_user = "root"
var db_pw = "password"
var game_points = [4]int{10, 9, 8, 7}
var timeserver_1_conn *websocket.Conn
var timeserver_2_conn *websocket.Conn

func main() {
	var portRead = -5
	log.Printf("Please give the port number that the server will be servicing on (between 8001 and 8100) :.\n")
	_, scanErr := fmt.Scan(&portRead)
	for scanErr != nil || portRead < 8001 || portRead > 8100 {
		if scanErr == nil {
			log.Printf("Port number for game service must be between 8001 and 8100.\n")
		} else {
			log.Print("Scan for port failed, due to error : ", scanErr.Error())
		}
		_, scanErr = fmt.Scan(&portRead)
		log.Printf("Please give the port number that the server will be servicing on (between 8001 and 8100):.\n")
	}
	GAME_SERVICE.port = strconv.Itoa(portRead)
	log.Printf("Read Port # : %d.\n", portRead)
	db1, err := sql.Open("mysql", db_user+":"+db_pw+"@tcp("+nginx_master.host+":"+nginx_master.port+")/mydb")
	if err != nil {
		log.Printf("There was an DSN issue when opening the DB driver. Error : %s.\n", err.Error())
	}
	db1.SetConnMaxLifetime(time.Minute * 3)
	db1.SetMaxOpenConns(10)
	db1.SetMaxIdleConns(10)

	defer func(db1 *sql.DB) {
		err := db1.Close()
		if err != nil {
			log.Printf("There was an error closing the database connection. Error : %s.\n", err.Error())
		}
	}(db1)

	db2, err := sql.Open("mysql", db_user+":"+db_pw+"@tcp("+nginx_slave.host+":"+nginx_slave.port+")/mydb")
	if err != nil {
		log.Printf("There was an DSN issue when opening the DB driver. Error : %s.\n", err.Error())
	}
	db2.SetConnMaxLifetime(time.Minute * 3)
	db2.SetMaxOpenConns(10)
	db2.SetMaxIdleConns(10)

	defer func(db2 *sql.DB) {
		err := db2.Close()
		if err != nil {
			log.Printf("There was an error closing the database connection. Error : %s.\n", err.Error())
		}
	}(db2)

	timeserver_1_conn = connectToTimeServer(TIME_SERVER_1)
	timeserver_2_conn = connectToTimeServer(TIME_SERVER_2)

	if connectToProxy() {
		//Listen for Game Service Requests if you were successfully registered at the Proxy
		log.Println("Successful registration to proxy.")
		gameServiceTCPAddr, err := net.ResolveTCPAddr(GAME_SERVICE.con_type, GAME_SERVICE.host+":"+GAME_SERVICE.port)
		if err != nil {
			log.Printf("Unable to resolve Address for Listening for game connections : %s:%s error : %s\n", GAME_SERVICE.host, GAME_SERVICE.port, err.Error())
			//If we can't resolve address there is not much we can do on the server side. Might as well just shut er' down.s
			os.Exit(1)
		}
		listener, err := net.ListenTCP("tcp", gameServiceTCPAddr)
		if err != nil {
			log.Printf("Unable to start listener - at address : %s:%s, %s", GAME_SERVICE.host, GAME_SERVICE.port, err)
		} else {
			log.Printf("Listening on %v:%v\n", GAME_SERVICE.host, GAME_SERVICE.port)
		}
		//close Listener when the go routine is over
		//Will see if we can decide on a mechanism to start and shut down servers gracefully later.
		defer func(listener *net.TCPListener) {
			err := listener.Close()
			if err != nil {
				log.Printf("There was an error closing the listener connection. Error : %s.\n", err.Error())
			}
		}(listener)

		//Continuously Listen for game Connections
		for {
			//infinite listening - blocks while waiting in this go routine
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Printf("There was an error in Accepting the connection. Error : %s\n", err.Error())
				//Error with listener? Should we read from keyboard for IP Address and port to listen to ?
			}
			log.Printf("Incoming connection from : %s\n", conn.RemoteAddr().String())
			//Sub routine is called and we pass to it the connection parameter to be handled
			go handleGameConnection(db1,db2, conn)
		}
	} else {
		log.Println("Failed to register server at the proxy.")
	}
}

// Returns true if the proxy accepts the connection.
func connectToProxy() bool {
	//connection type, IpAddres:Port
	proxyAddr, err := net.ResolveTCPAddr(PROXY.con_type, PROXY.host+":"+PROXY.port)
	if err != nil {
		log.Println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}
	//attempt to connect to proxy using a tcp connection
	conn, err := net.DialTCP(PROXY.con_type, nil, proxyAddr)
	if err != nil {
		log.Println("Dial failed:", err.Error())
		os.Exit(1)
	}

	_, err = conn.Write([]byte("Server Join"))
	if err != nil {
		log.Println("Write data failed:", err.Error())
		os.Exit(1)
	}

	// buffer to get data
	received := make([]byte, 8192)
	n, err := conn.Read(received)
	if err != nil {
		log.Println("Read data failed:", err.Error())
		os.Exit(1)
	}
	if strings.Compare(string(received[:n]), "Accepted") == 0 {
		//If the proxy accepted us, send the address we will be serving at
		log.Printf("Received message: %s.\n", string(received[:n]))
		//Create a string IpAddress:PortNumber
		var gameServiceAddress = GAME_SERVICE.host + ":" + GAME_SERVICE.port
		_, err = conn.Write([]byte(gameServiceAddress))
		if err != nil {
			log.Println("Write data failed:", err.Error())
			os.Exit(1)
		} else {
			//wait for acknowledgement.
			n, err = conn.Read(received)
			if err != nil {
				log.Println("Read data failed:", err.Error())
				os.Exit(1)
			} else {
				//proxy received address success.
				if strings.Compare(string(received[:n]), "Received Address") == 0 {
					log.Printf("Received message: %s.\n", string(received[:n]))
					conn.Close()
					return true
				}
			}
		}
	}
	conn.Close()
	return false
}

func testConnection(db *sql.DB, host string, port string) bool {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	err := db.PingContext(ctx)
	if err != nil {
		log.Printf("Errors : %s, while pinging DB", err)
		return false
	} else {
		log.Printf("Successfully accessed database at %s:%s.\n", host, port)
		return true
	}
}

// Returns true if the proxy accepts the connection.
func connectToTimeServer(time_server connection) *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: time_server.host+":"+time_server.port, Path: "/ws/server",}
    log.Printf("connecting to %s", u.String())
    conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil);

    if err != nil {
        log.Printf("handshake failed with status %d", resp.StatusCode)
        log.Fatal("dial:", err)
    }
	err = conn.WriteMessage(1, []byte("Server Join"))
	if err != nil {
		log.Println("Write data failed:", err.Error())
		os.Exit(1)
	}

	// buffer to get data
	received := make([]byte, 8192)
	_, received, err = conn.ReadMessage()
	if err != nil {
		log.Println("Read data failed:", err.Error())
		os.Exit(1)
	}
	if strings.Compare(string(received[:len(received)]), "Accepted") == 0 {
		//If the proxy accepted us, send the address we will be serving at
		log.Printf("The message from time server is Accepted.")
	}
	return conn
}
