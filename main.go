package main

import (
	"fmt"
	"ipfs-alpha-entanglement-code/cmd"
	ipfscluster "ipfs-alpha-entanglement-code/ipfs-cluster"
	"ipfs-alpha-entanglement-code/util"
	"os"
)

func main() {
	util.Enable_LogPrint()

	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			// os.Exit(1)
		}
	}()

	alpha, s, p := 3, 5, 5
	path := "test/data/largeFile.txt"

	err := cmd.Upload(path, alpha, s, p)
	util.CheckError(err, "fail uploading file %s or its entanglement", path)

	/* Simple ipfs cluster test, with GET request */
	ipfscluster, _ := ipfscluster.CreateIPFSClusterConnector(0)
	result, err := ipfscluster.PeerLs()
	util.CheckError(err, "fail to execute IPFS cluster peer ls")
	util.LogPrint(util.Yellow(result))

	result, err = ipfscluster.PeerInfo()
	util.CheckError(err, "fail to execute IPFS cluster peer info")
	util.LogPrint(util.Green(result))

}
