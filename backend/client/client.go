//Author         : Apostolos Scondrianis
//Created On     : 01-03-2023
//Last Edited By : Apostolos Scondrianis
//Last Edit On   : 02-03-2023
//Filename       : client.go
//Version        : 0.1

// client to emulate functionality of web client
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

type connection struct {
	host     string
	port     string
	con_type string
}

// Address that the proxy listens for clients
var PROXY = connection{"127.0.0.1", "8000", "tcp"}

var USERNAME string = "testUser"

func main() {
	//Create TCP address to connect to proxy
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
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Simple Shell")
		fmt.Println("---------------------")
		for {
			fmt.Print("-> ")
			text, _ := reader.ReadString('\n')
			// convert CRLF to LF
			text = strings.Replace(text, "\r\n", "", -1)
			var command []string = strings.Split(string(text), ":")
			if strings.Compare(command[0], "Create Room") == 0 {
				createRoom(conn, text)
			} else if strings.Compare(command[0], "Exit") == 0 {
				break
			}
		}
		conn.Close()
		break
	}
}

func createRoom(conn net.Conn, request string) {
	_, err := conn.Write([]byte(request))
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
	fmt.Printf("Message Received: %s\n", string(received[:n]))
}
