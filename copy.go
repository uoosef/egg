package main

import (
	"egg/wsconnadapter"
	"fmt"
	"io"
)

func chanFromConn(reader io.Reader) chan []byte {
	c := make(chan []byte)

	go func() {
		b := make([]byte, 1024)

		for {
			n, err := reader.Read(b)
			if n > 0 {
				res := make([]byte, n)
				// Copy the buffer so it doesn't get changed while read by the recipient.
				copy(res, b[:n])
				c <- res
			}
			if err != nil {
				fmt.Println(err)
				c <- nil
				break
			}
		}
	}()

	return c
}

// Copy accepts a websocket connection and TCP connection and copies data between them
func Copy(wsConn *wsconnadapter.Adapter, reader io.Reader, writer io.Writer) {
	wsChan := chanFromConn(wsConn)
	tcpChan := chanFromConn(reader)

	for {
		select {
		case wsData := <-wsChan:
			if wsData == nil {
				fmt.Println("websocket channel closed")
				return
			} else {
				writer.Write(wsData)
			}
		case tcpData := <-tcpChan:
			if tcpData == nil {
				fmt.Printf("remote tcp channel closed")
				return
			} else {
				wsConn.Write(tcpData)
			}
		}
	}

}
