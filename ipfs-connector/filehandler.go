package ipfsconnector

import (
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"
	"os"
	"strings"

	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

func (c *IPFSConnector) UploadFile(path string, alpha int, s int, p int) error {
	// add original file to ipfs
	cid, err := c.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	fmt.Printf(util.White("Finish adding file to IPFS with CID %s. File path: %s\n"), cid.String(), path)

	// get merkle tree from swarm and flattern the tree
	root, err := c.GetMerkleTree(cid)
	util.CheckError(err, "could not read merkle tree")
	nodes := root.GetFlattenedTree(s, p)
	util.LogPrint(util.Green("Number of nodes in the merkle tree is %d. Node sequence:"), len(nodes))
	for _, node := range nodes {
		util.LogPrint(util.Green(" %d"), node.PostOrderIdx)
	}
	util.LogPrint("\n")
	fmt.Println(util.White("Finish reading file's merkle tree from IPFS"))

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
	fmt.Println(util.White("Finish generating entanglement"))

	// write entanglement to files and upload to ipfs
	entanglementFilenamePrefix := strings.Split(path, ".")[0]
	for k, parities := range entanglement {
		entanglementFilename := fmt.Sprintf("%s_entanglement_%d", entanglementFilenamePrefix, k)
		err = os.WriteFile(entanglementFilename, parities, 0644)
		util.CheckError(err, "fail to write entanglement file")
		cid, err := c.AddFile(entanglementFilename)
		util.CheckError(err, "could not add entanglement file to IPFS")
		fmt.Printf(util.White("Finish adding entanglement to IPFS with CID %s. File path: %s\n"), cid.String(), entanglementFilename)
	}

	return nil
}

// AddFile takes the file in the given path and writes it to IPFS network
func (c *IPFSConnector) AddFile(path string) (path.Resolved, error) {
	// prepare file
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	filenode, err := files.NewSerialFile(path, false, stat)
	if err != nil {
		return nil, err
	}

	// add the file node to IPFS
	cid, err := c.Unixfs().Add(c.ctx, filenode)
	if err != nil {
		return nil, err
	}

	return cid, nil
}

// GetFile takes the file CID and reads it from IPFS network
func (c *IPFSConnector) GetFile(cid path.Resolved, outputPath string) error {
	// get file node from IPFS
	rootNodeFile, err := c.Unixfs().Get(c.ctx, cid)
	if err != nil {
		return err
	}

	// write to the output path
	err = files.WriteTo(rootNodeFile, outputPath)

	return err
}

// GetFileByBlocks takes the file CID and returns a slice of blocks at the leaves of the Merkle tree
func (c *IPFSConnector) GetFileByBlocks(cid path.Resolved) error {
	// get the cid node from the IPFS
	rootNodeFile, err := c.ResolveNode(c.ctx, cid)
	if err != nil {
		return err
	}

	nodeStat, err := rootNodeFile.Stat()
	if err != nil {
		return err
	}
	fmt.Println(util.Red(nodeStat.DataSize), cid.String(), len(rootNodeFile.Links()))

	// Iterate all links that this block points to
	for _, link := range rootNodeFile.Links() {
		err := c.GetFileByBlocks(path.IpfsPath(link.Cid))
		if err != nil {
			return err
		}
	}
	return nil
}

// GetMerkleTree takes the Merkle tree root CID, constructs the tree and returns the root node
func (c *IPFSConnector) GetMerkleTree(cid path.Resolved) (*TreeNode, error) {
	currIdx := 0
	var getMerkleNode func(path.Resolved) (*TreeNode, error)
	getMerkleNode = func(cid path.Resolved) (*TreeNode, error) {
		// get the cid node from the IPFS
		rootNodeFile, err := c.ResolveNode(c.ctx, cid)
		if err != nil {
			return nil, err
		}

		rootNode := CreateTreeNode(rootNodeFile.RawData())

		// iterate all links that this block points to
		for _, link := range rootNodeFile.Links() {
			childNode, err := getMerkleNode(path.IpfsPath(link.Cid))
			if err != nil {
				return nil, err
			}
			rootNode.AddChild(childNode)
		}

		// update node idx
		rootNode.PostOrderIdx = currIdx
		currIdx++

		return rootNode, nil
	}

	return getMerkleNode(cid)
}
