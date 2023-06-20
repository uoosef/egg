package statute

import "egg/bufferpool"

type NetworkType int32

const (
	TCP NetworkType = 0
	UDP NetworkType = 1
)

type PathType int32

const (
	Upload   PathType = 0
	Download PathType = 1
	TwoWay   PathType = 2
)

type CommandType int32

const (
	StartFlow    CommandType = 0
	ContinueFlow CommandType = 1
	EndFlow      CommandType = 2
)

var (
	RelayAddress          string = ""
	RelayAddressToReplace string = ""
	BufferPool            bufferpool.BufPool
)
