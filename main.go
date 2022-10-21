package main

import (
	"fmt"

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
	// tangler := entangler.Entangler{Alpha: 3, S: 5, P: 5, ChunkSize: 1048}
	root, err := connector.GetMerkleTree(cid)
	if err != nil {
		panic(fmt.Errorf("could not read merkle tree: %s", err))
	}
	nodes := root.GetFlattenedTree(5, 5)
	fmt.Printf(util.White("Number of nodes in the merkle tree is %d"), len(nodes))
}
