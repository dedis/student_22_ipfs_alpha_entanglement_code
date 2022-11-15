package cmd

import (
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"strings"
)

// Upload uploads the original file, generates and uploads the entanglement of that file
func (c *Client) Upload(path string, alpha int, s int, p int) error {
	conn, err := ipfsconnector.CreateIPFSConnector(0)
	util.CheckError(err, "failed to spawn peer node")

	// add original file to ipfs
	roodCID, err := conn.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	util.LogPrint("Finish adding file to IPFS with CID %s. File path: %s", roodCID, path)

	if alpha < 1 {
		// expect no entanglement
		return nil
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
	err = tangler.WriteEntanglementToFile(maxSize, outputPaths)
	util.CheckError(err, "fail to write entanglement to file")
	util.LogPrint("Finish generating entanglement")

	// upload entanglements to ipfs
	for _, entanglementFilename := range outputPaths {
		cid, err := conn.AddFile(entanglementFilename)
		util.CheckError(err, "could not add entanglement file to IPFS")
		util.LogPrint("Finish adding entanglement to IPFS with CID %s. File path: %s", cid, entanglementFilename)
	}

	// Store Metatdata?
	CIDIndexMap := make(map[string]int)
	for i, node := range nodes {
		CIDIndexMap[node.CID] = i
	}
	metaData := Metadata{
		Alpha:       alpha,
		S:           s,
		P:           p,
		CIDIndexMap: CIDIndexMap,
	}
	c.AddMetaData(roodCID, &metaData)

	return nil
}
