package ipfsconnector

import (
	"bytes"
	"fmt"
	"math"
	"os"

	"ipfs-alpha-entanglement-code/entangler"

	sh "github.com/ipfs/go-ipfs-api"
	"github.com/ipfs/kubo/core/coredag"
)

// IPFSConnector manages all the interaction with IPFS node
type IPFSConnector struct {
	shell *sh.Shell
}

var Default_Port int = 5001

// CreateIPFSConnector creates a running IPFS node and returns a connector to it
func CreateIPFSConnector(port int) (*IPFSConnector, error) {
	if port == 0 {
		port = Default_Port
	}
	return &IPFSConnector{sh.NewShell(fmt.Sprintf("localhost:%d", port))}, nil
}

// AddFile takes the file in the given path and writes it to IPFS network
func (c *IPFSConnector) AddFile(path string) (cid string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	return c.shell.Add(file)
}

// GetFile takes the file CID and reads it from IPFS network
func (c *IPFSConnector) GetFile(cid string, outputPath string) error {
	return c.shell.Get(cid, outputPath)
}

// AddRawData addes raw block data to IPFS network
func (c *IPFSConnector) AddRawData(chunk []byte) (cid string, err error) {
	return c.shell.BlockPut(chunk, "v0", "sha2-256", -1)
}

// GetRawBlock gets raw block data from IPFS network
func (c *IPFSConnector) GetRawBlock(cid string) (data []byte, err error) {
	return c.shell.BlockGet(cid)
}

// GetLinksFromRawBlock extracts the links from the internal node
func (c *IPFSConnector) GetLinksFromRawBlock(chunk []byte) (CIDs []string, err error) {
	ipldnodes, err := coredag.ParseInputs("raw", "dag-pb", bytes.NewReader(chunk), math.MaxUint64, -1)
	if err != nil {
		return
	}	
	for _, link := range ipldnodes[0].Links() {
		CIDs = append(CIDs, link.Cid.String())
	}
	return
}

// GetMerkleTree takes the Merkle tree root CID, constructs the tree and returns the root node
func (c *IPFSConnector) GetMerkleTree(cid string, lattice *entangler.Lattice) (*TreeNode, error) {
	currIdx := 0
	var getMerkleNode func(string) (*TreeNode, error)

	getMerkleNode = func(cid string) (*TreeNode, error) {
		// get the cid node from the IPFS
		rootNodeFile, err := c.shell.ObjectGet(cid) //c.api.ResolveNode(c.ctx, cid)
		if err != nil {
			return nil, err
		}

		rootNode := CreateTreeNode([]byte{})
		rootNode.CID = cid
		rootNode.connector = c
		// update node idx
		rootNode.PreOrderIdx = currIdx
		currIdx++

		// iterate all links that this block points to
		if len(rootNodeFile.Links) > 0 {
			for _, link := range rootNodeFile.Links {
				childNode, err := getMerkleNode(link.Hash)
				if err != nil {
					return nil, err
				}
				rootNode.AddChild(childNode)
			}
		} else {
			rootNode.LeafSize = 1
		}

		return rootNode, nil
	}

	return getMerkleNode(cid)
}
