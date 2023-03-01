//Author         : Apostolos Scondrianis
//Created On     : 28-02-2023
//Last Edited By : Apostolos Scondrianis
//Last Edit On   : 28-02-2023
//Filename       : proxy.go
//Version        : 0.1

package main

//Proxy
import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)
type server struct {
	host string;
	port string;
	con_type string;
}

var (
	mu sync.Mutex
	servers_slice[] server
)

func main() {
	// Resolve TCP Address
	var proxyInfo = server {host:"127.0.0.1", port:"9000", con_type:"tcp"}
	//Address to be listening on
	proxyAddr, err := net.ResolveTCPAddr(proxyInfo.con_type, proxyInfo.host+":"+proxyInfo.port)
	if err != nil {
		fmt.Printf("Unable to resolve IP")
	}

	// Start TCP Listener
	listener, err := net.ListenTCP("tcp", proxyAddr)
	if err != nil {
		fmt.Printf("Unable to start listener - %s", err)
	} else {
		fmt.Printf("Listening on %v:%v\n", proxyInfo.host, proxyInfo.port)
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
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	// incoming request


	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	if strings.Compare(string(buffer[:n]), "Server Join") == 0 {
		//Client attempting to connect is a server
		host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			fmt.Println(err)
		}
		//lock the servers_slice variable

		time := time.Now().Format(time.ANSIC)
		fmt.Printf("Command : %v. Send Accepted.\n", string(buffer[:]))
		//responseStr := fmt.Sprintf("Accepted")
		conn.Write([]byte("Accepted"))
		n, err = conn.Read(buffer)
		if err != nil {
			log.Fatal(err)
		}
		host, port, err = net.SplitHostPort(string(buffer[:n]))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("")
			mu.Lock()
			servers_slice = append(servers_slice, server{host:host, port:port, con_type:"tcp"})
			mu.Unlock()
			fmt.Printf("%s was added as a server on the server list on %v.\n", string(buffer[:n]), time)
			conn.Write([]byte("Received Address"))
		}
	} else {
		conn.Write([]byte("Your registration as a server is declined."))
	}
	// close conn
	conn.Close()
}