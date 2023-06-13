package main

import (
	"bytes"
	"egg/wsconnadapter"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"net"
	"net/http"
)

type Server struct {
	cp *ConnectionPool
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (sf *Server) get(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("new get request")
	_, err := io.WriteString(w, "This is my website!\n")
	if err != nil {
		return
	}
}
func (sf *Server) ws(w http.ResponseWriter, r *http.Request) {
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		fmt.Printf("HTTP SERVER, WS Connection Upgrade: %v", err.Error())
		return
	}
	conn := wsconnadapter.New(wsConn)

	size := make([]byte, 2)
	conn.Read(size)

	fmt.Println(size)

	data := make([]byte, binary.BigEndian.Uint16(size))
	conn.Read(data)

	var reqAck bytes.Buffer
	reqAck.Write(data)
	dec := gob.NewDecoder(&reqAck) // Will read from network.
	var q PathReq
	err = dec.Decode(&q)
	if err != nil {
		return
	}

	fmt.Println("connecting to", q.Dest, "...")
	defer fmt.Println("connection to", q.Dest, "closed !")

	var destConn net.Conn

	destConn, found := sf.cp.GetSrvConnection(q.Id)

	if !found {
		// connect to remote server
		netType := "tcp"
		if q.Net == UDP {
			netType = "udp"
		}

		destConn, err = net.Dial(netType, q.Dest)
		if err != nil {
			fmt.Println("unable to connect to" + q.Dest + " " + err.Error())
			return
		}

		if q.PType != TwoWay {
			sf.cp.NewSrvConnection(q.Id, destConn)
		}
	}

	errCh := make(chan error, 2)

	// upload path
	if q.PType == Upload || q.PType == TwoWay {
		go func() { errCh <- Copy(conn, destConn) }()
	}

	// download path
	if q.PType == Download || q.PType == TwoWay {
		go func() { errCh <- Copy(destConn, conn) }()
	}

	// Wait
	err = <-errCh
	if err != nil {
		fmt.Println("transport error:", err)
	}

	destConn.Close()
	conn.Close()
}

func (sf *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", sf.ws)
	mux.HandleFunc("/", sf.get)

	return http.ListenAndServe(addr, mux)
}

func NewServer() *Server {
	cp := NewConnectionPool()
	return &Server{
		cp,
	}
}
