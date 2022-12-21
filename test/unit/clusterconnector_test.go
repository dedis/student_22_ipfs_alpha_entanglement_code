package test

import (
	"fmt"
	ipfscluster "ipfs-alpha-entanglement-code/ipfs-cluster"
	"ipfs-alpha-entanglement-code/util"
	"testing"
)

func Test_Cluster_Simple_Info(t *testing.T) {
	util.EnableLogPrint()
	for i := 0; i < 10; i++ {
		/* Connect to different peers */
		ipfscluster, _ := ipfscluster.CreateIPFSClusterConnector(9094 + i*100)
		peerName, err := ipfscluster.PeerInfo()
		if err != nil {
			t.Fatal("fail to execute IPFS cluster peer info: ", err)
		}
		util.LogPrintf(fmt.Sprintf("Connected IPFS Cluster peer: %s", peerName))

		nbPeer, err := ipfscluster.PeerLs()
		if err != nil {
			t.Fatal("fail to execute IPFS cluster peer ls: ", err)
		}
		util.LogPrintf(fmt.Sprintf("Number of IPFS Cluster peers: %d", nbPeer))
	}
}

func Test_Cluster_Pin(t *testing.T) {
	util.EnableLogPrint()
	ipfscluster, _ := ipfscluster.CreateIPFSClusterConnector(9094)
	cid1 := "QmQqzMTavQgT4f4T5v6PWBp7XNKtoPmC9jvn12WPT3gkSE"
	cid2 := "bafkreidlgzgnujigow46cy6t6pru23hqcox5agypq7sala6fnvq4ggo4zu"
	replicationFactor := 1
	err := ipfscluster.AddPin(cid1, replicationFactor)
	if err != nil {
		t.Fatalf("fail to execute IPFS cluster peer pin %s: %s\n", cid1, err)
	}
	util.LogPrintf(fmt.Sprintf("Pin new cid: %s", cid1))
	err = ipfscluster.AddPin(cid2, replicationFactor)
	if err != nil {
		t.Fatalf("fail to execute IPFS cluster peer pin %s: %s\n", cid2, err)
	}
	util.LogPrintf(fmt.Sprintf("Pin new cid: %s", cid2))
}

func Test_Cluster_Pin_Info(t *testing.T) {
	util.EnableLogPrint()
	ipfscluster, _ := ipfscluster.CreateIPFSClusterConnector(9094)
	pinStatus, err := ipfscluster.PinStatus("")
	if err != nil {
		t.Fatal("fail to execute IPFS cluster peer pin status: ", err)
	}
	util.LogPrintf(fmt.Sprintf("Pinned files: %s", pinStatus))
}

func Test_Cluster_Load_Check(t *testing.T) {
	util.EnableLogPrint()
	ipfscluster, _ := ipfscluster.CreateIPFSClusterConnector(9094)
	peerLoad, err := ipfscluster.PeerLoad()
	if err != nil {
		t.Fatal("fail to execute IPFS cluster peer load: ", err)
	}
	util.LogPrintf(fmt.Sprintf("Load on peers: %s", peerLoad))
}
