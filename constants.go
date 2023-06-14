package main

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

var (
	RelayAddress          string = ""
	RelayAddressToReplace string = ""
	BufferPool            bufferpool.BufPool
)
