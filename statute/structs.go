package statute

import (
	"context"
	"io"
	"net"
)

type SocksRequest struct {
	ID          string // A unique ID to identify a connection
	Network     NetworkType
	Ctx         context.Context
	Dest        string
	Writer      io.Writer
	Reader      io.Reader
	CloseSignal chan error
}

type ServerConnection struct {
	Id   string // A unique Id to identify a connection
	Conn net.Conn
}

type PathReq struct {
	Id    string
	Dest  string
	Net   NetworkType
	PType PathType
}

type PacketHeader struct {
	Id          string
	Command     CommandType
	PayloadSize int
	Sequence    int64
}

type QueuePacket struct {
	Header *PacketHeader
	Body   []byte
	Err    error
}
