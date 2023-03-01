//Author         : Apostolos Scondrianis
//Created On     : 28-02-2023
//Last Edited By : Apostolos Scondrianis
//Last Edit On   : 28-02-2023
//Filename       : proxy.go
//Version        : 0.1

package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type connection struct {
	host string;
	port string;
	con_type string;
}
var  PROXY = connection{"127.0.0.1", "9000", "tcp"}
var SERVER = connection{"127.0.0.1", "8080", "tcp"}

func main() {
	if(connectToProxy()) {
		fmt.Println("Successful registration to proxy.")
	} else {
		fmt.Println("Failed to register server at the proxy.")
	}
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
		var serverAddr = SERVER.host+":"+SERVER.port
		_, err = conn.Write([]byte(serverAddr))
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