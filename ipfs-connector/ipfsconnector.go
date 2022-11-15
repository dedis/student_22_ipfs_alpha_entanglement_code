package ipfsconnector

import (
	"fmt"
	"os"

	sh "github.com/ipfs/go-ipfs-api"

	"ipfs-alpha-entanglement-code/entangler"
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
