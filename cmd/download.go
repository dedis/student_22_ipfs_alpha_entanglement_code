package cmd

import (
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"os"

	"golang.org/x/xerrors"
)

type DownloadOption struct {
	MetaCID           string
	UploadRecoverData bool
	DataFilter        map[int]struct{}
}

// Download download the original file, repair it if metadata is provided
func (c *Client) Download(rootCID string, path string, option DownloadOption) (err error) {
	// direct downloading if no metafile provided
	if len(option.MetaCID) == 0 {
		fmt.Println(err)
		// try to down original file using given rootCID (i.e. no metafile)
		err = c.GetFile(rootCID, path)
		if err != nil {
			return xerrors.Errorf("fail to download original file: %s", err)
		}
		util.LogPrint("Finish downloading file (no recovery)")

		return nil
	}

	// download metafile
	// TODO: lazy downloading?
	metaData, err := c.GetMetaData(option.MetaCID)
	if err != nil {
		return xerrors.Errorf("fail to download metaData: %s", err)
	}
	util.LogPrint("Finish downloading metaFile")

	chunkNum := len(metaData.DataCIDIndexMap)
	// create getter
	getter := ipfsconnector.CreateIPFSGetter(c.IPFSConnector, metaData.DataCIDIndexMap, metaData.ParityCIDs)
	getter.DataFilter = option.DataFilter
	// create lattice
	lattice := entangler.NewLattice(metaData.Alpha, metaData.S, metaData.P, chunkNum, getter)
	lattice.Init()
	util.LogPrint("Finish generating lattice")

	// download & recover file from IPFS
	data := []byte{}
	repaired := false
	var walker func(string) error
	walker = func(cid string) (err error) {
		chunk, hasRepaired, err := lattice.GetChunk(metaData.DataCIDIndexMap[cid])
		if err != nil {
			return xerrors.Errorf("fail to recover chunk with CID: %s", err)
		}

		// upload missing chunk back to the network if allowed
		repaired = repaired || hasRepaired
		if option.UploadRecoverData && hasRepaired {
			uploadCID, err := c.AddRawData(chunk)
			if err != nil || uploadCID != cid {
				return xerrors.Errorf("fail to upload the repaired chunk to IPFS: %s", err)
			}
		}

		// unmarshal and iterate
		dagNode, err := c.GetDagNodeFromRawBytes(chunk)
		if err != nil {
			return xerrors.Errorf("fail to parse raw data: %s", err)
		}
		links := dagNode.Links()
		if len(links) > 0 {
			for _, link := range links {
				err = walker(link.Cid.String())
				if err != nil {
					return
				}
			}
		} else {
			fileChunkData, err := c.GetFileDataFromDagNode(dagNode)
			if err != nil {
				return xerrors.Errorf("fail to parse file data: %s", err)
			}
			data = append(data, fileChunkData...)
		}
		return
	}
	walker(metaData.RootCID)

	// write to file in the given path
	err = os.WriteFile(path, data, 0644)
	if err == nil {
		if repaired {
			util.LogPrint("Finish downloading file (recovered)")
		} else {
			util.LogPrint("Finish downloading file (no recovery)")
		}
	}

	return
}
