package cmd

import (
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"strings"
)

// Upload uploads the original file, generates and uploads the entanglement of that file
func (c *Client) Upload(path string, alpha int, s int, p int) (roodCID string, err error) {
	conn, err := ipfsconnector.CreateIPFSConnector(0)
	util.CheckError(err, "failed to connect to IPFS node")

	// add original file to ipfs
	roodCID, err = conn.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	util.LogPrint("Finish adding file to IPFS with CID %s. File path: %s", roodCID, path)

	if alpha < 1 {
		// expect no entanglement
		return
	}

	// get merkle tree from IPFS and flatten the tree
	root, err := conn.GetMerkleTree(roodCID, &entangler.Lattice{})
	util.CheckError(err, "could not read merkle tree")
	nodes := root.GetFlattenedTree(s, p, true)
	util.InfoPrint(util.Green("Number of nodes in the merkle tree is %d. Node sequence:"), len(nodes))
	for _, node := range nodes {
		util.InfoPrint(util.Green(" %d"), node.PreOrderIdx)
	}
	util.InfoPrint("\n")
	util.LogPrint("Finish reading and flattening file's merkle tree from IPFS")

	// generate entanglement
	data := make(chan []byte, len(nodes))
	maxSize := 0
	for _, node := range nodes {
		nodeData, err := node.Data()
		util.CheckError(err, "fail to load chunk data from IPFS")
		data <- nodeData
		if len(nodeData) > maxSize {
			maxSize = len(nodeData)
		}
	}
	close(data)
	tangler := entangler.NewEntangler(alpha, s, p)

	outputPaths := make([]string, alpha)
	for k := 0; k < alpha; k++ {
		outputPaths[k] = fmt.Sprintf("%s_entanglement_%d", strings.Split(path, ".")[0], k)
	}
	err = tangler.Entangle(data)
	util.CheckError(err, "fail to generate entanglement")

	// store parity blocks one by one
	parityCIDs := make([][]string, alpha)
	for k, parityBlocks := range tangler.ParityBlocks {
		cids := make([]string, 0)
		for i, block := range parityBlocks {
			blockCID, err := conn.AddRawData(block.Data)
			util.CheckError(err, "fail to upload entanglement %d on Strand %d", i, k)
			cids = append(cids, blockCID)
		}
		parityCIDs[k] = cids
		util.LogPrint("Finish generating entanglement %d", k)
	}

	// Store Metatdata?
	cidMap := make(map[string]int)
	for i, node := range nodes {
		cidMap[node.CID] = i + 1
	}
	metaData := Metadata{
		Alpha:           alpha,
		S:               s,
		P:               p,
		DataCIDIndexMap: cidMap,
		ParityCIDs:      parityCIDs,
	}
	c.AddMetaData(roodCID, &metaData)

	return
}
