package ipfsconnector

import (
	"fmt"
	"ipfs-alpha-entanglement-code/util"
	"os"

	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

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

func (c *IPFSConnector) GetFileByBlocks(cid path.Resolved) error {
	// get the cid node from the IPFS
	rootNodeFile, err := c.api.ResolveNode(c.ctx, cid)
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
		err = c.GetFileByBlocks(path.IpfsPath(link.Cid))
		if err != nil {
			return err
		}
	}

	return nil
}
