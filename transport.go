package main

import (
	"bytes"
	"egg/socks5"
	"egg/socks5/statute"
	"egg/wsconnadapter"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/gorilla/websocket"
)

/*func wsDialer(ctx context.Context, url string) (conn net.Conn, err error) {
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

		/*LSClient: func(conn net.Conn, hostname string) net.Conn {
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
}*/

func wsClient(socksReq *SocksReq, socksStream *Request, endpoint string, pathType PathType) {
	// connect to remote server via ws
	wsConn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
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

	Copy(conn, socksStream.reader, socksStream.writer)
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
