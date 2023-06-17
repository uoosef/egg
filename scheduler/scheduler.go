package scheduler

import (
	"egg/dial"
	"net"
)

/*
scheduler task is to get packets from client and write them to pre or post prepared network paths
for mux it should establish finite number of connections to server and watch them that they dont get destroyed then
relay the messages through them(network path must be antagonistic from actual "real" connections between client
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

func newPersistenceNetConn() {
	d := &dial.Dial{}
}

// sink is responsible for making actual network connections
func sink() {

}
