package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/gobwas/ws"
	"io"
	"net"
	"net/http"
)

type Server struct {
	cp *ConnectionPool
}

func (sf *Server) get(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("new get request")
	_, err := io.WriteString(w, "This is my website!\n")
	if err != nil {
		return
	}
}
func (sf *Server) ws(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		// handle error
	}
	go func() {
		defer conn.Close()

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

		destConn, found := sf.cp.GetSrvConnection(q.Dest)

		if !found {
			// connect to remote server
			netType := "tcp"
			if q.Net == UDP {
				netType = "udp"
			}

			destConn, err = net.Dial(netType, q.Dest)
			defer destConn.Close()
			if err != nil {
				fmt.Println("unable to connect to" + q.Dest + " " + err.Error())
				return
			}

			if q.PType != TwoWay {
				sf.cp.NewSrvConnection(q.Id, destConn)
			}
		}

		// Start proxying
		chanNums := 1
		if q.PType == TwoWay {
			chanNums = 2
		}
		errCh := make(chan error, chanNums)

		// upload path
		if q.PType == TwoWay || q.PType == Upload {
			go func() { errCh <- Copy(bufio.NewWriter(conn), bufio.NewReader(destConn)) }()
		}

		// download path
		if q.PType == TwoWay || q.PType == Download {
			go func() { errCh <- Copy(bufio.NewWriter(destConn), bufio.NewReader(conn)) }()
		}

		// Wait
		for i := 0; i < chanNums; i++ {
			e := <-errCh
			if e != nil {
				// return from this function closes target (and conn).
				fmt.Println("encountered error with read/write operation:", err)
				return
			}
		}
	}()
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
