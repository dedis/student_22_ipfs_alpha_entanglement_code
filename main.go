package main

import (
	"fmt"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
)

func main() {
	connector, err := ipfsconnector.CreateIPFSConnector(false)
	if err != nil {
		panic(fmt.Errorf("failed to spawn peer node: %s", err))
	}
	defer connector.Stop()

	// test add file
	cid, err := connector.AddFile("README.md")
	if err != nil {
		panic(fmt.Errorf("could not add File: %s", err))
	}
	fmt.Printf("Added file to IPFS with CID %s\n", cid.String())

	// test read file
	err = connector.GetFile(cid, "./test.md")
	if err != nil {
		panic(fmt.Errorf("could not get file with CID: %s", err))
	}
	fmt.Printf("Got file to IPFS with CID %s\n", cid.String())

}
