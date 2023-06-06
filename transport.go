package main

import (
	"bufio"
	"bytes"
	"context"
	"egg/socks5"
	"egg/socks5/statute"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/gobwas/ws"
	tls "github.com/refraction-networking/utls"
	"net"
	"time"
)

func wsDialer(ctx context.Context, url string) (conn net.Conn, err error) {
	dialer := ws.Dialer{
		NetDial: func(ctx context.Context, network, addr string) (net.Conn, error) {
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
		},

		TLSClient: func(conn net.Conn, hostname string) net.Conn {
			config := tls.Config{
				ServerName:             hostname,
				InsecureSkipTimeVerify: true,
				InsecureSkipVerify:     true,
			}
			return tls.UClient(conn, &config, tls.HelloRandomized)
		},
	}

	conn, _, _, err = dialer.Dial(ctx, url)
	return
}

func wsClient(socksReq *SocksReq, socksStream *Request, endpoint string, pathType PathType) {
	// connect to remote server via ws
	conn, err := wsDialer(context.Background(), endpoint)
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
	conn.Write(bs)

	conn.Write(sendBuffer.Bytes())

	// Start proxying
	chanNums := 1
	if pathType == TwoWay {
		chanNums = 2
	}
	errCh := make(chan error, chanNums)

	// upload path
	if pathType == TwoWay || pathType == Upload {
		go func() { errCh <- Copy(bufio.NewWriter(conn), socksStream.reader) }()
	}

	// download path
	if pathType == TwoWay || pathType == Download {
		go func() { errCh <- Copy(socksStream.writer, bufio.NewReader(conn)) }()
	}

	// Wait
	for i := 0; i < chanNums; i++ {
		e := <-errCh
		if e != nil {
			// return from this function closes target (and conn).
			socksStream.closeSignal <- err
			fmt.Println("encode error:", err)
			return
		}
	}

	err = conn.Close()
	if err != nil {
		socksStream.closeSignal <- err
		fmt.Println("encode error:", err)
		return
	}

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
