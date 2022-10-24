package main

import (
	"fmt"

	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
)

func main() {
	connector, err := ipfsconnector.CreateIPFSConnector(false)
	util.CheckError(err, "failed to spawn peer node")
	defer connector.Stop()

	// test add file
	cid, err := connector.AddFile("test/data/largeFile.txt")
	util.CheckError(err, "could not add File")
	fmt.Printf(util.White("Added file to IPFS with CID %s\n"), cid.String())

	// test read file
	//err = connector.GetFile(cid, "test.md")
	//util.CheckError(err, "could not get file with CID")
	//fmt.Printf("Got file to IPFS with CID %s\n", cid.String())

	// test read files by block
	// err = connector.GetFileByBlocks(cid)
	// util.CheckError(err, "could not get file by block with CID")
	// fmt.Printf(util.White("Read blocks in IPFS with CID %s\n"), cid.String())

	// test merkle tree
	root, err := connector.GetMerkleTree(cid)
	util.CheckError(err, "could not read merkle tree")
	nodes := root.GetFlattenedTree(1, 2)
	fmt.Printf(util.White("Number of nodes in the merkle tree is %d. Node sequence:"), len(nodes))
	for _, node := range nodes {
		fmt.Printf(util.Green(" %d"), node.PostOrderIdx)
	}
	fmt.Println()

	// test entanglement
	data := make([][]byte, len(nodes))
	maxSize := 0
	for i, node := range nodes {
		data[i] = node.Data
		if maxSize < len(node.Data) {
			maxSize = len(node.Data)
		}
	}
	tangler := entangler.NewEntangler(3, 5, 5, maxSize, &data)
	tangler.GetEntanglement()
}
