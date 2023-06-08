package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func NewRelay(bind string, server string) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp", bind)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn, server)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, server string) {
	client, err := net.Dial("tcp", server)
	if err != nil {
		log.Printf("Dial failed: %v", err)
		defer conn.Close()
		return
	}
	log.Printf("Forwarding from %v to %v\n", conn.LocalAddr(), client.RemoteAddr())
	//TODO
}
