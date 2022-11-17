package main

import (
	"fmt"
	"ipfs-alpha-entanglement-code/cmd"
	"ipfs-alpha-entanglement-code/util"
	"os"
)

func main() {
	util.Enable_LogPrint()
	// util.Enable_InfoPrint()

	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			// os.Exit(1)
		}
	}()

	alpha, s, p := 3, 5, 5
	path := "test/data/largeFile.txt"

	client := cmd.NewClient()
	cid, err := client.Upload(path, alpha, s, p)
	util.CheckError(err, "fail uploading file %s or its entanglement", path)

	err = client.Download(cid, "test/data/downloaded_largeFile.txt", true)
	util.CheckError(err, "fail downloading file %s", path)

	// /* Simple ipfs cluster test, with GET request */
	// ipfscluster, _ := ipfscluster.CreateIPFSClusterConnector(9094)
	// peerName, err := ipfscluster.PeerInfo()
	// util.CheckError(err, "fail to execute IPFS cluster peer info")
	// util.LogPrint(fmt.Sprintf("Connected IPFS Cluster peer: %s", peerName))

	// nbPeer, err := ipfscluster.PeerLs()
	// util.CheckError(err, "fail to execute IPFS cluster peer ls")
	// util.LogPrint(fmt.Sprintf("Number of IPFS Cluster peers: %d", nbPeer))

	// cid1 := "QmTy4FELeqWSZLdRehF5HdPeHUaA1uCU5YNf5A2zHxqiFn"
	// cid2 := "QmayFoFM47uNAxxZiibAYXBj2rMfivu2arwd9AhUCrXNDn"
	// err = ipfscluster.AddPin(cid1)
	// util.CheckError(err, "fail to execute IPFS cluster add pin")
	// util.LogPrint(fmt.Sprintf("Pin new cid: %s", cid1))
	// err = ipfscluster.AddPin(cid2)
	// util.CheckError(err, "fail to execute IPFS cluster add pin")
	// util.LogPrint(fmt.Sprintf("Pin new cid: %s", cid2))

	// time.Sleep(time.Second)

	// pinStatus, err := ipfscluster.PinStatus("")
	// util.CheckError(err, "fail to execute IPFS cluster pin status")
	// util.LogPrint(fmt.Sprintf("Pinned files: %s", pinStatus))
}
