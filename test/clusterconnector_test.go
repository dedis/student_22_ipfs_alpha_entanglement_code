package test

import (
	"fmt"
	ipfscluster "ipfs-alpha-entanglement-code/ipfs-cluster"
	"ipfs-alpha-entanglement-code/util"
	"testing"
	"time"
)

func Test_Cluster_Simple(t *testing.T) {
	util.Enable_LogPrint()
	ipfscluster, _ := ipfscluster.CreateIPFSClusterConnector(9094)
	peerName, err := ipfscluster.PeerInfo()
	if err != nil {
		t.Fatal("fail to execute IPFS cluster peer info: ", err)
	}
	util.LogPrint(fmt.Sprintf("Connected IPFS Cluster peer: %s", peerName))

	nbPeer, err := ipfscluster.PeerLs()
	if err != nil {
		t.Fatal("fail to execute IPFS cluster peer ls: ", err)
	}
	util.LogPrint(fmt.Sprintf("Number of IPFS Cluster peers: %d", nbPeer))

	cid1 := "QmQqzMTavQgT4f4T5v6PWBp7XNKtoPmC9jvn12WPT3gkSE"
	cid2 := "bafkreidlgzgnujigow46cy6t6pru23hqcox5agypq7sala6fnvq4ggo4zu"
	replicationFactor := 1
	err = ipfscluster.AddPin(cid1, replicationFactor)
	if err != nil {
		t.Fatalf("fail to execute IPFS cluster peer pin %s: %s\n", cid1, err)
	}
	util.LogPrint(fmt.Sprintf("Pin new cid: %s", cid1))
	err = ipfscluster.AddPin(cid2, replicationFactor)
	if err != nil {
		t.Fatalf("fail to execute IPFS cluster peer pin %s: %s\n", cid2, err)
	}
	util.LogPrint(fmt.Sprintf("Pin new cid: %s", cid2))

	time.Sleep(time.Second)

	pinStatus, err := ipfscluster.PinStatus("")
	if err != nil {
		t.Fatal("fail to execute IPFS cluster peer pin status: ", err)
	}
	util.LogPrint(fmt.Sprintf("Pinned files: %s", pinStatus))
}
