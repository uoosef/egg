package main

import (
	"context"
	"io"
	"net"
)

type ConnectionPool struct {
	cache *Cache
}

type Request struct {
	id          string // A unique id to identify a connection
	network     NetworkType
	ctx         context.Context
	writer      io.Writer
	reader      io.Reader
	closeSignal chan error
}

type ServerConnection struct {
	id   string // A unique id to identify a connection
	conn net.Conn
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		NewCache(0),
	}
}

func (cp *ConnectionPool) NewConnection(t NetworkType, closeSignal chan error, ctx context.Context, writer io.Writer, reader io.Reader) string {
	cID := NewUUID()
	cp.cache.Set(cID, Request{
		cID,
		t,
		ctx,
		writer,
		reader,
		closeSignal,
	})
	return cID
}

func (cp *ConnectionPool) NewSrvConnection(id string, conn net.Conn) {
	cp.cache.Set(id, ServerConnection{
		id,
		conn,
	})
}

func (cp *ConnectionPool) GetConnection(cID string) (Request, bool) {
	c, found := cp.cache.Get(cID)
	return c.(Request), found
}

func (cp *ConnectionPool) GetSrvConnection(cID string) (net.Conn, bool) {
	c, found := cp.cache.Get(cID)
	if !found {
		return nil, false
	}
	return c.(ServerConnection).conn, found
}

func (cp *ConnectionPool) RmConnection(cID string) {
	cp.cache.Delete(cID)
}
