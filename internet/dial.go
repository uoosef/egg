package internet

import (
	"context"
	"github.com/gorilla/websocket"
	tls "github.com/refraction-networking/utls"
	"net"
	"strings"
	"time"
)

type Connection struct {
	addr                   string
	overwriteAddr          string
	shouldOverWriteAddress bool
}

func Dial(addr, overwriteAddr string, shouldOverWriteAddress bool) (net.Conn, error) {
	c := &Connection{
		addr,
		overwriteAddr,
		shouldOverWriteAddress,
	}
	return c.wsDial()
}

func (d *Connection) plainTCPDial(ctx context.Context, network string) (net.Conn, error) {
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
	if d.shouldOverWriteAddress && strings.Contains(d.addr, d.overwriteAddr) {
		return dialer.DialContext(ctx, network, d.overwriteAddr)
	}
	return dialer.DialContext(ctx, network, d.addr)
}

func (d *Connection) wsDial() (net.Conn, error) {
	dialer := websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return d.plainTCPDial(ctx, network)
		},

		NetDialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			plainConn, err := d.plainTCPDial(ctx, network)
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

	conn, _, err := dialer.Dial(d.addr, nil)
	return newWsConnAdapter(conn), err
}

/*func (d *Connection) wsDial() (net.Conn, error) {
	wsConn, err := d.wsDialer()
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

	return wsConn), nil
}*/
