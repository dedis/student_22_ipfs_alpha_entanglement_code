package ipfsconnector

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"ipfs-alpha-entanglement-code/entangler"

	sh "github.com/ipfs/go-ipfs-api"
	dag "github.com/ipfs/go-merkledag"
	unixfs "github.com/ipfs/go-unixfs"
)

// IPFSConnector manages all the interaction with IPFS node
type IPFSConnector struct {
	shell *sh.Shell
}

var DefaultPort = 5001

// CreateIPFSConnector creates a running IPFS node and returns a connector to it
func CreateIPFSConnector(port int) (*IPFSConnector, error) {
	if port == 0 {
		port = DefaultPort
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

// AddFileFromMem takes the bytes array and upload it to IPFS network as file
func (c *IPFSConnector) AddFileFromMem(data []byte) (cid string, err error) {
	return c.shell.Add(bytes.NewReader(data))
}

// GetFile takes the file CID and reads it from IPFS network
func (c *IPFSConnector) GetFile(cid string, outputPath string) error {
	return c.shell.Get(cid, outputPath)
}

// GetFileToMem takes the file CID and reads it from IPFS network to memory
func (c *IPFSConnector) GetFileToMem(cid string) ([]byte, error) {
	data, err := c.shell.Cat(cid)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(data)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// AddRawData addes raw block data to IPFS network
func (c *IPFSConnector) AddRawData(chunk []byte) (cid string, err error) {
	return c.shell.BlockPut(chunk, "v0", "sha2-256", -1)
}

// GetRawBlock gets raw block data from IPFS network
func (c *IPFSConnector) GetRawBlock(cid string) (data []byte, err error) {
	return c.shell.BlockGet(cid)
}

// GetDagNodeFromRawBytes unmarshals raw bytes into IPFS dagnode
func (c *IPFSConnector) GetDagNodeFromRawBytes(chunk []byte) (dagnode *dag.ProtoNode, err error) {
	dagnode, err = dag.DecodeProtobuf(chunk)
	return
}

// GetFileDataFromDagNode extracts the real file data from IPFS dagnode
func (c *IPFSConnector) GetFileDataFromDagNode(dagnode *dag.ProtoNode) (data []byte, err error) {
	fsn, err := unixfs.FSNodeFromBytes(dagnode.Data())
	if err != nil {
		return
	}

	data = fsn.Data()
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

// GetTotalBlocks returns the total number of blocks in the DAG pointed by the cid
func (c *IPFSConnector) GetTotalBlocks(cid string) (int, error) {
	filestate, err := c.shell.FilesStat(context.Background(), "/ipfs/"+cid)
	if err != nil {
		return 0, err
	}
	return filestate.Blocks + 1, nil
}
