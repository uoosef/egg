package client

import (
	"context"
	"egg/socks5"
	socksStatute "egg/socks5/statute"
	"egg/statute"
	"egg/utils"
	"fmt"
	"io"
)

type Client struct {
	clientId     string
	endpoint     string
	relayEnabled bool
	muxEnabled   bool
}

func NewClient(endpoint string, relayEnabled bool, muxEnabled bool) (*socks5.Server, error) {
	clientId := utils.NewUUID()
	c := Client{
		clientId,
		endpoint,
		relayEnabled,
		muxEnabled,
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

	// it informs the socks transport that connection to remote host was successfully established
	if err := socks5.SendReply(writer, socksStatute.RepSuccess, nil); err != nil {
		return fmt.Errorf("failed to send reply, %v", err)
	}

	req := &statute.SocksReq{
		Id:   id,
		Dest: socksRequest.RawDestAddr.String(),
		Net:  netType,
	}

	if c.muxEnabled {
		go relayClient(req, &socksReq, endpoint)
	} else if c.relayEnabled {
		go relayConnect(req, &socksReq, endpoint)
	} else {
		go plainConnect(req, &socksReq, endpoint, statute.TwoWay)
	}

	// terminate the connection
	return <-closeSignal
}

// handle reader function, it sends a channel to scheduler and read from it and write its data to socks writer
