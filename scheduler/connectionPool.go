package scheduler

import (
	"context"
	"egg/statute"
	"egg/utils"
	"io"
	"net"
)

type ConnectionPool struct {
	cache *utils.Cache
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		utils.NewCache(0),
	}
}

func (cp *ConnectionPool) NewConnection(t statute.NetworkType, closeSignal chan error, ctx context.Context, writer io.Writer, reader io.Reader) string {
	cID := utils.NewUUID()
	cp.cache.Set(cID, statute.Request{
		ReqID:       cID,
		Network:     t,
		Ctx:         ctx,
		Writer:      writer,
		Reader:      reader,
		CloseSignal: closeSignal,
	})
	return cID
}

func (cp *ConnectionPool) NewSrvConnection(id string, conn net.Conn) {
	cp.cache.Set(id, statute.ServerConnection{
		Id:   id,
		Conn: conn,
	})
}

func (cp *ConnectionPool) GetConnection(cID string) (statute.Request, bool) {
	c, found := cp.cache.Get(cID)
	return c.(statute.Request), found
}

func (cp *ConnectionPool) GetSrvConnection(cID string) (net.Conn, bool) {
	c, found := cp.cache.Get(cID)
	if !found {
		return nil, false
	}
	return c.(statute.ServerConnection).Conn, found
}

func (cp *ConnectionPool) RmConnection(cID string) {
	cp.cache.Delete(cID)
}
