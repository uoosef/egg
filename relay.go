package main

import (
	"bufio"
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

	errCh := make(chan error, 2)

	go func() { errCh <- Copy(bufio.NewWriter(conn), bufio.NewReader(client)) }()
	go func() { errCh <- Copy(bufio.NewWriter(client), bufio.NewReader(conn)) }()

	// Wait
	for i := 0; i < 2; i++ {
		if <-errCh != nil {
			// return from this function closes target (and conn).
			fmt.Println("encode error:", err)
			return
		}
	}

	err = conn.Close()
	if err != nil {
		fmt.Println("encode error:", err)
		return
	}
}
