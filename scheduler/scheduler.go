package scheduler

import (
	"net"
)

/*
scheduler task is to get packets from client and write them to pre or post prepared network paths
for mux it should establish finite number of internet to server and watch them that they dont get destroyed then
relay the messages through them(network path must be antagonistic from actual "real" internet between client
and destination server
*/

type scheduler struct {
	schedulerType  string
	netConnections []*net.Conn
}

func NewScheduler(schedulerType string) *scheduler {
	return &scheduler{
		schedulerType: schedulerType,
	}
}

// sink is responsible for making actual network internet
func sink() {

}
