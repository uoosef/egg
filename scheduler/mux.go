package scheduler

import (
	"context"
	"egg/statute"
	"egg/tunnel"
	"egg/utils"
	"fmt"
	"io"
)

// mux scheduler is responsible for making finite number of persistent tunnels and
// move packets through them periodically

var (
	sendFifo    *utils.FIFO
	receiveFifo *utils.FIFO
)

type muxScheduler struct {
	tunnels          *utils.Cache
	tunnelsIDS       []string
	socksConnections *utils.Cache
	ctx              context.Context
	cancelCtx        context.CancelFunc
	cancelCtxCalled  bool
}

func NewMuxScheduler() *muxScheduler {
	ctx, cancelCtx := context.WithCancel(context.TODO())

	m := &muxScheduler{
		tunnels:          utils.NewCache(0),
		tunnelsIDS:       []string{},
		socksConnections: utils.NewCache(0),
		ctx:              ctx,
		cancelCtx:        cancelCtx,
		cancelCtxCalled:  false,
	}
	go m.RouteQueuePacketsThroughTunnels()
	go m.SendPacketsFromQueueToSocksConnections()
	return m
}

func (mux *muxScheduler) makeTunnels(
	size int, endpoint string,
	overwriteAddr string,
	shouldOverWriteAddress bool,
) {
	for i := 0; i < size; i++ {
		// make a tunnel
		tID := utils.NewUUID()
		mux.tunnelsIDS = append(mux.tunnelsIDS, tID)
		pt, err := tunnel.NewPersistentTunnel(tID, endpoint, overwriteAddr, shouldOverWriteAddress)
		if err != nil {
			fmt.Printf("Attempt to make tunnel %s failed with error: %s, retrying...\n", tID, err.Error())
			i--
			continue
		}
		mux.tunnels.Set(tID, pt)
	}
	fmt.Printf("Successfully made %d tunnels\n", size)
}

// RouteQueuePacketsThroughTunnels route packets through tunnels, it should run in separate go routine
func (mux *muxScheduler) RouteQueuePacketsThroughTunnels() {
	var i uint8
	for i = 0; ; i++ {
		// get packet from send queue
		packet, err := sendFifo.DequeueOrWaitForNextElementContext(mux.ctx)
		if packet == nil && mux.cancelCtxCalled {
			return
		}
		if err != nil {
			fmt.Printf("Error while dequeueing packet from send queue: %s\n", err.Error())
			continue
		}
		if packet == nil {
			continue
		}
		// get tunnel from tunnels
		tunnelID := mux.tunnelsIDS[i%uint8(len(mux.tunnelsIDS))]
		t, ok := mux.tunnels.Get(tunnelID)
		if !ok {
			fmt.Printf("Tunnel with id %s not found\n", tunnelID)
			continue
		}
		// send packet through tunnel
		err = t.(*tunnel.PersistentTunnel).Write(packet.(*statute.QueuePacket))
		if err != nil {
			return
		}
	}
}

// GetPacketsFromTunnelsToQueue get packets from tunnels and put them in the reception cache
func (mux *muxScheduler) GetPacketsFromTunnelsToQueue() {
	for _, tunnelID := range mux.tunnelsIDS {
		go func(tunnelID string) {
			for {
				t, ok := mux.tunnels.Get(tunnelID)
				if !ok {
					fmt.Printf("Tunnel with id %s not found\n", tunnelID)
					continue
				}
				packet, err := t.(*tunnel.PersistentTunnel).Read(mux.ctx)
				if packet == nil && mux.cancelCtxCalled {
					return
				}
				if err != nil {
					fmt.Printf("Error while reading from tunnel %s: %s\n", tunnelID, err.Error())
					continue
				}
				if packet == nil {
					continue
				}
				err = receiveFifo.Enqueue(packet)
				if err != nil {
					panic(err)
				}
			}
		}(tunnelID)
	}
}

// SendPacketsFromQueueToSocksConnections send packets from receive queue to registered socks connections
// it should write body of packets in socks connections writer
// if packet header indicated that connection is closed it should
// close the connection and remove it from cache
func (mux *muxScheduler) SendPacketsFromQueueToSocksConnections() {
	for {
		select {
		case <-mux.ctx.Done():
			return
		default:
			packet, err := receiveFifo.DequeueOrWaitForNextElementContext(mux.ctx)
			if packet == nil && mux.cancelCtxCalled {
				return
			}
			if err != nil || packet == nil {
				fmt.Printf("Error while dequeueing packet from receive queue: %s\n", err.Error())
				continue
			}
			// get socks connection
			cID := packet.(*statute.QueuePacket).Header.Id
			c, ok := mux.socksConnections.Get(cID)
			if !ok {
				fmt.Printf("Socks connection with id %s not found\n", cID)
				continue
			}
			// if packet header indicated that connection is closed then
			// close the connection and remove it from cache
			if packet.(*statute.QueuePacket).Header.Command == statute.EndFlow {
				// TODO: fix packet flow order(if some packet arrives faster than it should, cache it for certain
				// amount of time)
				c.(statute.Request).CloseSignal <- nil
				mux.socksConnections.Delete(cID)
				continue
			}
			// write packet body in socks connection writer
			_, err = c.(statute.Request).Writer.Write(packet.(*statute.QueuePacket).Body)
			if err != nil {
				fmt.Printf("Error while writing packet body in socks connection: %s\n", err.Error())
				continue
			}
		}
	}
}

// FromSocksConnectionToSendQueue read packets from registered socks connection and write them to send queue
func (mux *muxScheduler) FromSocksConnectionToSendQueue(flowId string, socksCtx context.Context, reader io.Reader) {
	buf := statute.BufferPool.Get()
	defer statute.BufferPool.Put(buf)
	command := statute.StartFlow
	for sequence := 0; ; sequence++ {
		select {
		case <-socksCtx.Done():
			// TODO: send close packet if connection is open
			return
		case <-mux.ctx.Done():
			return
		default:
			// read nr from socks connection
			nr, err := reader.Read(buf[:cap(buf)])
			if err != nil {
				fmt.Printf("Error while reading nr from socks connection: %s\n", err.Error())
				continue
			}
			if sequence > 0 {
				command = statute.ContinueFlow
			}
			queuePacket := &statute.QueuePacket{
				Header: &statute.PacketHeader{
					Id:          flowId,
					Command:     command,
					PayloadSize: nr,
					Sequence:    int64(sequence),
				},
				Body: buf[:nr],
				Err:  nil,
			}
			// write nr to send queue
			err = sendFifo.Enqueue(queuePacket)
			if err != nil {
				panic(err)
			}
		}
	}
}

// RegisterSocksConnection register a socks connection in cache and run the reader in separate goroutine
func (mux *muxScheduler) RegisterSocksConnection(t statute.NetworkType, closeSignal chan error, ctx context.Context, writer io.Writer, reader io.Reader, dest string) {
	cID := utils.NewUUID()
	mux.socksConnections.Set(cID, statute.Request{
		ReqID:       cID,
		Network:     t,
		Dest:        dest,
		Ctx:         ctx,
		Writer:      writer,
		Reader:      reader,
		CloseSignal: closeSignal,
	})
	go mux.FromSocksConnectionToSendQueue(cID, ctx, reader)
}
