package main

import (
	"fmt"

	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
)

func main() {
	connector, err := ipfsconnector.CreateIPFSConnector(false)
	if err != nil {
		panic(fmt.Errorf("failed to spawn peer node: %s", err))
	}
	defer connector.Stop()

	// test add file
	cid, err := connector.AddFile("test/largeFile.txt")
	if err != nil {
		panic(fmt.Errorf("could not add File: %s", err))
	}
	fmt.Printf(util.White("Added file to IPFS with CID %s\n"), cid.String())

	// test read file
	//err = connector.GetFile(cid, "test.md")
	//if err != nil {
	//	panic(fmt.Errorf("could not get file with CID: %s", err))
	//}
	//fmt.Printf("Got file to IPFS with CID %s\n", cid.String())

	// test read files by block
	// err = connector.GetFileByBlocks(cid)
	// if err != nil {
	// 	panic(fmt.Errorf("could not get file by block with CID: %s", err))
	// }
	// fmt.Printf(util.White("Read blocks in IPFS with CID %s\n"), cid.String())

	// test merkle tree
	root, err := connector.GetMerkleTree(cid)
	if err != nil {
		panic(fmt.Errorf("could not read merkle tree: %s", err))
	}
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
	tangler := entangler.NewEntangler(3, 5, 5, maxSize)
	tangler.Entangle(data)
}
