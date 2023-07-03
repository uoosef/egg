package client

import (
	"context"
	"egg/scheduler"
	"egg/socks5"
	socksStatute "egg/socks5/statute"
	"egg/statute"
	"egg/utils"
	"fmt"
	"io"
)

type Client struct {
	clientId  string
	scheduler *scheduler.Scheduler
}

func NewClient(endpoint string) (*socks5.Server, error) {
	clientId := utils.NewUUID()
	sc := scheduler.NewScheduler()
	c := Client{
		clientId,
		sc,
	}
	s5 := socks5.NewServer(
		socks5.WithConnectHandle(func(ctx context.Context, writer io.Writer, request *socks5.Request) error {
			return c.handle(ctx, writer, request, statute.TCP)
		}),
		socks5.WithAssociateHandle(func(ctx context.Context, writer io.Writer, request *socks5.Request) error {
			return c.handle(ctx, writer, request, statute.UDP)
		}),
	)
	return s5, nil
}

func (c *Client) handle(socksCtx context.Context, writer io.Writer, socksRequest *socks5.Request, netType statute.NetworkType) error {
	fmt.Println(socksRequest.RawDestAddr)
	closeSignal := make(chan error)

	/* it informs the socks transport the connection to remote host was successfully established
	Generally for connection establishment, we have to options:
	1- send a connect request to server and wait for its response
	in this method server can connect to remote address and provide us with detailed information about the connection
	such as domain resolving was successful or not, or the connection was established successfully or not
	2- trick the socks client and send a reply to it that connection was established successfully, it will help that
	the successfully established connections have one less round trip, and it will be faster, but it has a drawback
	because the client doesn't know anything about the connection, for example if the domain resolving would be unsuccessful
	or any other error occurred, the client will only get a reset error, and it will be confusing for the user
	*/
	if err := socks5.SendReply(writer, socksStatute.RepSuccess, nil); err != nil {
		return fmt.Errorf("failed to send reply, %v", err)
	}

	/*
		every connection has a unique id, it is used to identify the connection. in actual tcp or udp connections, the
		unique id is the combination of source ip, source port, destination ip and destination port, but in this case
		in sake of simplicity we use a random uuid as the unique id
	*/
	id := utils.NewUUID()

	req := &statute.SocksRequest{
		ID:          id,
		Network:     netType,
		Ctx:         socksCtx,
		Dest:        socksRequest.RawDestAddr.String(),
		Writer:      writer,
		Reader:      socksRequest.Reader,
		CloseSignal: closeSignal,
	}

	c.scheduler.RegisterSocksConnection(req)

	// terminate the connection
	return <-closeSignal
}

// handle reader function, it sends a channel to scheduler and read from it and write its data to socks writer
