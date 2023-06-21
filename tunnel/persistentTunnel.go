package tunnel

import (
	"bytes"
	"context"
	"egg/internet"
	"egg/statute"
	"egg/utils"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type PersistentTunnel struct {
	id                     string
	conn                   net.Conn
	readMutex              sync.Mutex
	writeMutex             sync.Mutex
	sendQueue              *utils.FIFO
	receiveQue             *utils.FIFO
	ctx                    context.Context
	cancelCtx              context.CancelFunc
	endpoint               string
	overwriteAddr          string
	shouldOverWriteAddress bool
	tunnelSendSequence     int
	tunnelReceiveSequence  int
	lastPacketToSend       []byte
	reconnecting           bool
	connectionIsClosed     bool
}

/*
	unpack response and get the command.
	command types:
	1- establish --> server's concern
	2- continue  --> scheduler's concern
	3- shutdown --> client's concern
*/

func NewPersistentTunnel(id, addr, overwriteAddr string, shouldOverWriteAddress bool) (*PersistentTunnel, error) {
	ctx, cancelCtx := context.WithCancel(context.TODO())

	conn := &PersistentTunnel{
		id:                     id,
		ctx:                    ctx,
		cancelCtx:              cancelCtx,
		endpoint:               addr,
		overwriteAddr:          overwriteAddr,
		shouldOverWriteAddress: shouldOverWriteAddress,
		tunnelSendSequence:     0,
		tunnelReceiveSequence:  0,
	}
	fmt.Printf("attemping to connect to tunnel end point: %s", addr)
	err := conn.ReDial()
	if err == nil {
		conn.RunRoutines()
		return conn, nil
	}
	return nil, err
}

func (tunnel *PersistentTunnel) ReDial() error {
	var conn net.Conn
	var err error
	for retries := 0; retries < 10; retries++ {
		conn, err = internet.Dial(tunnel.endpoint, tunnel.overwriteAddr, tunnel.shouldOverWriteAddress)
		if err == nil {
			tunnel.conn = conn
			return nil
		} else {
			fmt.Printf("Unable to connect to tunnel server error: %v\r\n", err)
		}
	}
	return errors.New(fmt.Sprintf(
		"Unable to connect to %s maximum retries exceeded, wraping up...",
		tunnel.endpoint))
}

func (tunnel *PersistentTunnel) RunRoutines() {
	go tunnel.readFromNetworkConnection()
	go tunnel.writeToNetworkConnection()
}

// read from send queue and write to actual connection
func (tunnel *PersistentTunnel) writeToNetworkConnection() {
	for {
		select {
		case <-tunnel.ctx.Done():
			return
		default:
			// if tunnel last packet to write is not null write it to connection
			// instead of crafting new packet then make it null otherwise craft
			// new packet and send it through actual connection and if it encounters error
			// the fill the last packet and continue
			if tunnel.lastPacketToSend != nil {
				nw, err := tunnel.conn.Write(tunnel.lastPacketToSend)
				if len(tunnel.lastPacketToSend) == nw {
					tunnel.tunnelSendSequence++
					tunnel.lastPacketToSend = nil
				}
				if err != nil {
					continue
				}
			}
			_queuePacket, err := tunnel.sendQueue.Dequeue()
			if err != nil {
				continue
			}
			queuePacket := _queuePacket.(*statute.QueuePacket)
			if queuePacket.Err != nil {
				continue
			}
			err = tunnel.craftPacket(queuePacket)
			if err != nil {
				panic(err)
			}
			nw, err := tunnel.conn.Write(tunnel.lastPacketToSend)
			if len(tunnel.lastPacketToSend) == nw {
				tunnel.tunnelSendSequence++
				tunnel.lastPacketToSend = nil
			}
			if err != nil {
				tunnel.ReDial()
				continue
			}
		}
	}
}

// read from actual connection and write to receive Queue
func (tunnel *PersistentTunnel) readFromNetworkConnection() {
	for {
		select {
		case <-tunnel.ctx.Done():
			tunnel.connectionIsClosed = true
			return
		default:
			tunnelPacketHeader, packetBody, nr, er := tunnel.readPacket(tunnel.conn)
			if er != nil && !tunnel.connectionIsClosed {
				tunnel.ReDial()
				continue
			}
			if nr > 0 {
				ew := tunnel.receiveQue.Enqueue(&statute.QueuePacket{
					Header: tunnelPacketHeader,
					Body:   packetBody,
					Err:    er,
				})
				// increase tunnel receive sequence +1
				tunnel.tunnelReceiveSequence++
				if ew != nil {
					panic("Memory error! unable to enqueue new object")
				}
			}
			if tunnel.connectionIsClosed {
				return
			}
		}
	}
}

func (tunnel *PersistentTunnel) craftPacket(queuePacket *statute.QueuePacket) error {
	// 2 byte header size, header, data
	/*	command := statute.ContinueFlow
		if tunnel.tunnelSendSequence == 0 {
			command = statute.StartFlow
		}*/
	var packetHeader bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&packetHeader) // Will write to network.
	// Encode (send) the value.
	err := enc.Encode(queuePacket.Header)
	if err != nil {
		return errors.New("unable to encode packet header struct to bytes")
	}

	bs := make([]byte, 2)
	binary.BigEndian.PutUint16(bs, uint16(len(packetHeader.Bytes())))
	tunnel.lastPacketToSend = append(append(bs, packetHeader.Bytes()...), queuePacket.Body...)
	return nil
}

func (tunnel *PersistentTunnel) readPacket(conn net.Conn) (*statute.PacketHeader, []byte, int, error) {
	// part1: 2 byte header size, part2: header, part3: data
	// header is a struct that converts to go by gob
	_p1 := make([]byte, 2)
	tunnel.conn.SetReadDeadline(time.Now())
	read, err := conn.Read(_p1)
	if err != nil || read < 2 {
		if err == io.EOF {
			return nil, nil, 0, errors.New("closed connection")
		}
		return nil, nil, 0, errors.New("packet decompress error unable to get header size")
	}
	headerSize := binary.BigEndian.Uint16(_p1)
	_p2 := make([]byte, headerSize)
	_, err = conn.Read(_p2)
	if err != nil {
		return nil, nil, 0, errors.New("packet decompress error unable to read packet header bytes from network")
	}
	var _tmp bytes.Buffer
	_tmp.Write(_p2)
	dec := gob.NewDecoder(&_tmp) // Will read from network.
	var tunnelPacketHeader statute.PacketHeader
	err = dec.Decode(&tunnelPacketHeader)
	if err != nil {
		return nil, nil, 0, errors.New("packet decompress error unable to read packet header bytes from network")
	}
	packetBody := make([]byte, tunnelPacketHeader.PayloadSize)
	_, err = conn.Read(packetBody)
	if err != nil {
		return nil, nil, 0, errors.New("packet decompress error unable to read packet bytes from network")
	}
	return &tunnelPacketHeader, packetBody, len(packetBody), nil
}

func (tunnel *PersistentTunnel) Read(ctx context.Context) (*statute.QueuePacket, error) {
	tunnel.readMutex.Lock()
	defer tunnel.readMutex.Unlock()

	_p, err := tunnel.receiveQue.DequeueOrWaitForNextElementContext(ctx)
	p := _p.(*statute.QueuePacket)
	if err != nil {
		return nil, err
	}

	return p, err
}

func (tunnel *PersistentTunnel) Write(p *statute.QueuePacket) error {
	tunnel.writeMutex.Lock()
	defer tunnel.writeMutex.Unlock()

	err := tunnel.sendQueue.Enqueue(p)

	return err
}

func (tunnel *PersistentTunnel) Close() error {
	tunnel.cancelCtx()
	tunnel.connectionIsClosed = true
	return tunnel.conn.Close()
}

func (tunnel *PersistentTunnel) LocalAddr() net.Addr {
	return tunnel.conn.LocalAddr()
}

func (tunnel *PersistentTunnel) RemoteAddr() net.Addr {
	return tunnel.conn.RemoteAddr()
}

func (tunnel *PersistentTunnel) SetDeadline(t time.Time) error {
	if err := tunnel.SetReadDeadline(t); err != nil {
		return err
	}

	return tunnel.SetWriteDeadline(t)
}

func (tunnel *PersistentTunnel) SetReadDeadline(t time.Time) error {
	return tunnel.conn.SetReadDeadline(t)
}

func (tunnel *PersistentTunnel) SetWriteDeadline(t time.Time) error {
	return tunnel.conn.SetWriteDeadline(t)
}
