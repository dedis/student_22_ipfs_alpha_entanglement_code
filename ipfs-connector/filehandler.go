package ipfsconnector

import (
	"os"

	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

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
	cid, err := c.api.Unixfs().Add(c.ctx, filenode)
	if err != nil {
		return nil, err
	}

	return cid, nil
}

func (c *IPFSConnector) GetFile(cid path.Resolved, outputPath string) error {
	// get file node from IPFS
	rootNodeFile, err := c.api.Unixfs().Get(c.ctx, cid)
	if err != nil {
		return err
	}

	// write to the output path
	err = files.WriteTo(rootNodeFile, outputPath)

	return err
}
