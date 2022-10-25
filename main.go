package main

import (
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
)

func main() {
	util.Enable_LogPrint()

	connector, err := ipfsconnector.CreateIPFSConnector(false)
	util.CheckError(err, "failed to spawn peer node")
	defer connector.Stop()

	alpha, s, p := 3, 5, 5
	path := "test/data/largeFile.txt"

	err = connector.Upload(path, alpha, s, p)
	util.CheckError(err, "fail uploading file %s or its entanglement", path)
}
