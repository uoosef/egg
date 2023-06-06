package main

import (
	"fmt"
)

func Scheduler(fifo *FIFO, cp *ConnectionPool, endpoint string, relayEnabled bool) {
	for {
		fmt.Println("waiting for new element in queue")
		r, err := fifo.DequeueOrWaitForNextElement()
		if err != nil {
			panic(err)
		}
		req := r.(*SocksReq)
		socksReq, found := cp.GetConnection(req.Id)
		if !found {
			panic("the connection with following connection id missing: " + req.Id)
		}
		if relayEnabled {
			go relayClient(req, &socksReq, endpoint)
		} else {
			go wsClient(req, &socksReq, endpoint, TwoWay)
		}
	}
}
