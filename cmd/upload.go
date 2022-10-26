package cmd

import (
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"os"
	"strings"
)

// Upload uploads the original file, generates and uploads the entanglement of that file
func Upload(path string, alpha int, s int, p int) error {
	c, err := ipfsconnector.CreateIPFSConnector(0)
	util.CheckError(err, "failed to spawn peer node")

	// add original file to ipfs
	cid, err := c.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	util.LogPrint("Finish adding file to IPFS with CID %s. File path: %s", cid, path)

	if alpha < 1 {
		// expect no entanglement
		return nil
	}

	// get merkle tree from swarm and flattern the tree
	root, err := c.GetMerkleTree(cid)
	util.CheckError(err, "could not read merkle tree")
	nodes := root.GetFlattenedTree(s, p)
	util.InfoPrint(util.Green("Number of nodes in the merkle tree is %d. Node sequence:"), len(nodes))
	for _, node := range nodes {
		util.InfoPrint(util.Green(" %d"), node.PostOrderIdx)
	}
	util.InfoPrint("\n")
	util.LogPrint("Finish reading and flatterning file's merkle tree from IPFS")

	// generate entanglement
	data := make([][]byte, len(nodes))
	maxSize := 0
	for i, node := range nodes {
		data[i] = node.Data
		if maxSize < len(node.Data) {
			maxSize = len(node.Data)
		}
	}
	tangler := entangler.NewEntangler(alpha, s, p, maxSize, &data)
	entanglement := tangler.GetEntanglement()
	util.LogPrint("Finish generating entanglement")

	// write entanglement to files and upload to ipfs
	entanglementFilenamePrefix := strings.Split(path, ".")[0]
	for k, parities := range entanglement {
		entanglementFilename := fmt.Sprintf("%s_entanglement_%d", entanglementFilenamePrefix, k)
		err = os.WriteFile(entanglementFilename, parities, 0644)
		util.CheckError(err, "fail to write entanglement file")
		cid, err := c.AddFile(entanglementFilename)
		util.CheckError(err, "could not add entanglement file to IPFS")
		util.LogPrint("Finish adding entanglement to IPFS with CID %s. File path: %s", cid, entanglementFilename)
	}

	return nil
}
