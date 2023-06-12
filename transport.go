package main

import (
	"bytes"
	"context"
	"egg/socks5"
	"egg/socks5/statute"
	"egg/wsconnadapter"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/gorilla/websocket"
	tls "github.com/refraction-networking/utls"
	"net"
	"strings"
	"time"
)

var portMap = map[string]string{
	"http":  "80",
	"https": "443",
}

func plainTCPDial(ctx context.Context, network, addr string) (net.Conn, error) {
	var (
		dnsResolverIP        = "8.8.8.8:53" // Google DNS resolver.
		dnsResolverProto     = "udp"        // Protocol to use for the DNS resolver
		dnsResolverTimeoutMs = 5000         // Timeout (ms) for the DNS resolver (optional)
	)

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
				}
				return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
			},
		},
	}
	if ShouldRelayed && addr == RelayAddressToReplace {
		addr = RelayAddress
	}
	return dialer.DialContext(ctx, network, addr)
}

func wsDialer(address string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		NetDialContext: plainTCPDial,

		NetDialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			plainConn, err := plainTCPDial(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			config := tls.Config{
				ServerName:         strings.Split(addr, ":")[0],
				InsecureSkipVerify: true,
			}
			utlsConn := tls.UClient(plainConn, &config, tls.HelloAndroid_11_OkHttp)
			err = utlsConn.Handshake()
			if err != nil {
				_ = plainConn.Close()
				return nil, err
			}
			return utlsConn, nil
		},
	}

	conn, _, err := dialer.Dial(address, nil)
	return conn, err
}

func wsClient(socksReq *SocksReq, socksStream *Request, endpoint string, pathType PathType) {
	// connect to remote server via ws
	wsConn, err := wsDialer(endpoint)
	if err != nil {
		if err := socks5.SendReply(socksStream.writer, statute.RepServerFailure, nil); err != nil {
			socksStream.closeSignal <- err
			return
		}
		socksStream.closeSignal <- err
		fmt.Printf("Can not connect: %v\n", err)
		return
	}

	fmt.Printf("%s connected\n", socksReq.Id)

	conn := wsconnadapter.New(wsConn)

	pathReq := PathReq{
		socksReq.Id,
		socksReq.Dest,
		socksReq.Net,
		pathType,
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
	go func() { errCh <- Copy(socksStream.reader, conn) }()

	// download path
	go func() { errCh <- Copy(conn, socksStream.writer) }()

	// Wait
	err = <-errCh
	if err != nil {
		fmt.Println("transport error:", err)
	}

	conn.Close()
	socksStream.closeSignal <- nil
}

func relayClient(socksReq *SocksReq, socksStream *Request, endpoint string) {
	// connect to remote server via ws for upload
	ShouldRelayed = true
	wsClient(socksReq, socksStream, endpoint, Upload)

	// connect to remote server via ws for download
	ShouldRelayed = false
	wsClient(socksReq, socksStream, endpoint, Download)
}
