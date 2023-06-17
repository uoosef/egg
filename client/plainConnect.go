package client

import (
	"bytes"
	"egg/dial"
	socksStatute "egg/socks5/statute"
	"egg/statute"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strings"
)

func plainConnect(socksReq *statute.SocksReq, socksStream *socksStatute.Request, endpoint string, pathType statute2.PathType) {

	pathReq := statute.PathReq{
		Id:    socksReq.Id,
		Dest:  socksReq.Dest,
		Net:   socksReq.Net,
		PType: pathType,
	}

	var sendBuffer bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&sendBuffer) // Will write to network.
	// Encode (send) the value.
	err = enc.Encode(&pathReq)
	if err != nil {
		socksStream.closeSignal <- err
		fmt.Println("encode error:", err)
		return
	}

	bs := make([]byte, 2)
	binary.BigEndian.PutUint16(bs, uint16(len(sendBuffer.Bytes())))

	// writing request block size(2 bytes)
	conn.Write(append(bs, sendBuffer.Bytes()...))

	errCh := make(chan error, 2)

	// upload path
	if pathType == statute2.Upload || pathType == statute2.TwoWay {
		go func() { errCh <- dial.Copy(socksStream.reader, conn) }()
	}

	// download path
	if pathType == statute2.Download || pathType == statute2.TwoWay {
		go func() { errCh <- dial.Copy(conn, socksStream.writer) }()
	}

	// Wait
	err = <-errCh
	if err != nil && !strings.Contains(err.Error(), "websocket: close 1006") {
		fmt.Println("transport error:", err)
	}

	conn.Close()
	socksStream.closeSignal <- nil
}
