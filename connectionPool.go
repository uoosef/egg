package main

import (
	"context"
	"egg/socks5"
	"io"
	"net"
	"net/http"
)

type ConnectionPool struct {
	cache *Cache
}

type NetworkType int32

const (
	TCP NetworkType = 0
	UDP NetworkType = 1
)

type Request struct {
	id          string // A unique id to identify a connection
	network     NetworkType
	ctx         context.Context
	writer      io.Writer
	request     *socks5.Request
	closeSignal chan error
}

type ServerConnection struct {
	id      string // A unique id to identify a connection
	network NetworkType
	writer  *http.ResponseWriter
	conn    *net.TCPConn
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		NewCache(0),
	}
}

func (cp *ConnectionPool) NewConnection(t NetworkType, closeSignal chan error, ctx context.Context, writer io.Writer, request *socks5.Request) string {
	cID := NewUUID()
	cp.cache.Set(cID, Request{
		cID,
		t,
		ctx,
		writer,
		request,
		closeSignal,
	})
	return cID
}

func (cp *ConnectionPool) NewSrvConnection(id string, t NetworkType, writer *http.ResponseWriter, conn *net.TCPConn) {
	cp.cache.Set(id, ServerConnection{
		id,
		t,
		writer,
		conn,
	})
}

func (cp *ConnectionPool) GetConnection(cID string) (Request, bool) {
	c, found := cp.cache.Get(cID)
	return c.(Request), found
}

func (cp *ConnectionPool) GetSrvConnection(cID string) (ServerConnection, bool) {
	c, found := cp.cache.Get(cID)
	return c.(ServerConnection), found
}

func (cp *ConnectionPool) RmConnection(cID string) {
	cp.cache.Delete(cID)
}
