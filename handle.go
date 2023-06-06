package main

import (
	"context"
	"egg/socks5"
	"egg/socks5/statute"
	"fmt"
	"io"
)

type Handle struct {
	cp   *ConnectionPool
	fifo *FIFO
}

type SocksReq struct {
	Id   string
	Dest string
	Net  NetworkType
}

type PathReq struct {
	Id    string
	Dest  string
	Net   NetworkType
	PType PathType
}

func (c *Handle) handleTCPConnect(ctx context.Context, writer io.Writer, request *socks5.Request) error {
	fmt.Println(request.RawDestAddr)
	closeSignal := make(chan error)
	id := c.cp.NewConnection(TCP, closeSignal, ctx, writer, request.Reader)

	// it informs the socks client that connection to remote host was successfully established
	if err := socks5.SendReply(writer, statute.RepSuccess, nil); err != nil {
		return fmt.Errorf("failed to send reply, %v", err)
	}

	err := c.fifo.Enqueue(&SocksReq{
		id,
		request.RawDestAddr.String(),
		TCP,
	})

	if err != nil {
		return err
	}

	// terminate the connection
	return <-closeSignal
}

func (c *Handle) handleUDPAssociate(ctx context.Context, writer io.Writer, request *socks5.Request) error {
	fmt.Println(request.RawDestAddr)
	closeSignal := make(chan error)
	id := c.cp.NewConnection(UDP, closeSignal, ctx, writer, request.Reader)

	// it informs the socks client that connection to remote host was successfully established
	if err := socks5.SendReply(writer, statute.RepSuccess, nil); err != nil {
		return fmt.Errorf("failed to send reply, %v", err)
	}

	err := c.fifo.Enqueue(&SocksReq{
		id,
		request.RawDestAddr.String(),
		UDP,
	})

	if err != nil {
		return err
	}

	// terminate the connection
	return <-closeSignal
}
