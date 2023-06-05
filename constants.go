package main

type NetworkType int32

const (
	TCP NetworkType = 0
	UDP NetworkType = 1
)

type ConnectionType int32

const (
	Upload   ConnectionType = 0
	Download ConnectionType = 1
	TwoWay   ConnectionType = 2
)

var (
	RelayAddress          string = ""
	RelayAddressToReplace string = ""
)
