package main

import (
	"egg/socks5"
)

func NewClient(endpoint string) (*socks5.Server, error) {
	fifo := NewFIFO()
	cp := NewConnectionPool()
	h := Handle{
		cp,
		fifo,
	}
	s5 := socks5.NewServer(
		socks5.WithConnectHandle(h.handleTCPConnect),
		socks5.WithAssociateHandle(h.handleUDPAssociate),
	)
	go Scheduler(fifo, cp, endpoint)
	return s5, nil
}
