package statute

import (
	"context"
	"io"
	"net"
)

type Request struct {
	ReqID       string // A unique ReqID to identify a connection
	Network     NetworkType
	Ctx         context.Context
	Writer      io.Writer
	Reader      io.Reader
	CloseSignal chan error
}

type ServerConnection struct {
	Id   string // A unique Id to identify a connection
	Conn net.Conn
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

type NetResp struct {
	CID   string
	Bytes []byte
}
