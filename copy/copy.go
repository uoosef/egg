package copy

import (
	"egg/statute"
	"egg/utils"
	"errors"
	"io"
)

func CopyFromFifo(src *utils.FIFO, dst io.Writer) (err error) {
	for {
		bI, er := src.DequeueOrWaitForNextElement()
		buf := bI.([]byte)
		nr := len(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = errors.New("short write")
				break
			}
		}
		if er != nil {
			if er != errors.New("EOF") {
				err = er
			}
			break
		}
	}
	return err
}

func CopyToFifo(src io.Reader, dst *utils.FIFO, connectionId string) (err error) {
	buf := statute.BufferPool.Get()
	defer statute.BufferPool.Put(buf)

	for {
		nr, er := src.Read(buf[:cap(buf)])
		if nr > 0 {
			ew := dst.Enqueue(&statute.TunnelPacketHeader{
				CID:   connectionId,
				Bytes: buf[:nr],
			})
			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != errors.New("EOF") {
				err = er
			}
			break
		}
	}
	return err
}
