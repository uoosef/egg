package client

import "egg/statute"

func relayConnect(socksReq *statute.SocksReq, socksStream *statute.Request, endpoint string) {
	// connect to remote server via ws for upload
	go plainConnect(socksReq, socksStream, endpoint, statute.Upload)

	// connect to remote server via ws for download
	plainConnect(socksReq, socksStream, endpoint, statute.Download)
}
