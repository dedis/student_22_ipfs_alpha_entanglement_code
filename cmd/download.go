package cmd

import (
	"bytes"
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
	DataFilter        []int
	ParityFilter      []int
}

// Download download the original file, repair it if metadata is provided
func (c *Client) Download(rootCID string, path string, option DownloadOption) (out string, err error) {
	err = c.InitIPFSConnector()
	if err != nil {
		return "", err
	}

	// direct downloading if no metafile provided
	if len(option.MetaCID) == 0 {
		fmt.Println(err)
		// try to down original file using given rootCID (i.e. no metafile)
		err = c.GetFile(rootCID, path)
		if err != nil {
			return "", xerrors.Errorf("fail to download original file: %s", err)
		}
		util.LogPrint("Finish downloading file (no recovery)")

		return "", nil
	}

	// download metafile
	// TODO: lazy downloading?
	metaData, err := c.GetMetaData(option.MetaCID)
	if err != nil {
		return "", xerrors.Errorf("fail to download metaData: %s", err)
	}
	util.LogPrint("Finish downloading metaFile")

	chunkNum := len(metaData.DataCIDIndexMap)
	// create getter
	getter := ipfsconnector.CreateIPFSGetter(c.IPFSConnector, metaData.DataCIDIndexMap, metaData.ParityCIDs)
	if len(option.DataFilter) > 0 {
		getter.DataFilter = make(map[int]struct{}, len(option.DataFilter))
		for _, index := range option.DataFilter {
			getter.DataFilter[index] = struct{}{}
		}
		getter.ParityFilter = make(map[int]struct{}, len(option.ParityFilter))
		for _, index := range option.ParityFilter {
			getter.ParityFilter[index] = struct{}{}
		}
	}

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
			// TODO: does trimming zero always works?
			chunk = bytes.Trim(chunk, "\x00")
			uploadCID, err := c.AddRawData(chunk)
			if err != nil {
				return xerrors.Errorf("fail to upload the repaired chunk to IPFS: %s", err)
			}
			if uploadCID != cid {
				fmt.Printf(util.Magenta("%d, %d\n"), metaData.DataCIDIndexMap[cid], len(chunk))
				return xerrors.Errorf("incorrect CID of the repaired chunk. Expected: %s, Got: %s", cid, uploadCID)
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
					return err
				}
			}
		} else {
			fileChunkData, err := c.GetFileDataFromDagNode(dagNode)
			if err != nil {
				return xerrors.Errorf("fail to parse file data: %s", err)
			}
			data = append(data, fileChunkData...)
		}
		return err
	}
	err = walker(metaData.RootCID)
	if err != nil {
		return "", err
	}

	// write to file in the given path
	if len(path) == 0 {
		out = rootCID
	} else {
		out = path
	}

	err = os.WriteFile(out, data, 0644)
	if err == nil {
		if repaired {
			util.LogPrint("Finish downloading file (recovered)")
		} else {
			util.LogPrint("Finish downloading file (no recovery)")
		}
	}

	return out, nil
}
