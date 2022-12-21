package cmd

import (
	"bytes"
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
}

// Download download the original file, repair it if metadata is provided
func (c *Client) Download(rootCID string, path string, option DownloadOption) (out string, err error) {
	err = c.InitIPFSConnector()
	if err != nil {
		return "", err
	}

	/* direct downloading if no metafile provided */
	if len(option.MetaCID) == 0 {
		return c.directDownload(rootCID, path)
	}
	return c.metaDownload(rootCID, path, option)
}

// directDownload interacts directly with IPFS. It fails when any data is missing
func (c *Client) directDownload(rootCID string, path string) (out string, err error) {
	// try to down original file using given rootCID (i.e. no metafile)
	err = c.GetFile(rootCID, path)
	if err != nil {
		return "", xerrors.Errorf("fail to download original file: %s", err)
	}
	util.LogPrintf("Finish downloading file (no recovery)")

	return "", nil
}

// downloadAndRecover interacts with IPFS through lattice, It launches recovery if any data is missing
func (c *Client) downloadAndRecover(lattice *entangler.Lattice, metaData *Metadata,
	option DownloadOption) (data []byte, repaired bool, err error) {

	data = []byte{}
	repaired = false
	var walker func(string) error
	walker = func(cid string) (err error) {
		chunk, hasRepaired, err := lattice.GetChunk(metaData.DataCIDIndexMap[cid])
		if err != nil {
			return xerrors.Errorf("fail to recover chunk with CID: %s", err)
		}

		// upload missing chunk back to the network if allowed
		if hasRepaired {
			// Problem: does trimming zero always works?
			chunk = bytes.Trim(chunk, "\x00")
			err = c.dataReupload(chunk, cid, option.UploadRecoverData)
			if err != nil {
				return err
			}
		}
		repaired = repaired || hasRepaired

		// unmarshal and iterate
		dagNode, err := c.GetDagNodeFromRawBytes(chunk)
		if err != nil {
			return xerrors.Errorf("fail to parse raw data: %s", err)
		}
		links := dagNode.Links()
		for _, link := range links {
			err = walker(link.Cid.String())
			if err != nil {
				return err
			}
		}

		if len(links) == 0 {
			fileChunkData, err := c.GetFileDataFromDagNode(dagNode)
			if err != nil {
				return xerrors.Errorf("fail to parse file data: %s", err)
			}
			data = append(data, fileChunkData...)
		}
		return err
	}
	err = walker(metaData.RootCID)
	return data, repaired, err
}

// metaDownload download metadata for recovery usage
func (c *Client) metaDownload(rootCID string, path string, option DownloadOption) (out string, err error) {
	/* download metafile */
	metaData, err := c.GetMetaData(option.MetaCID)
	if err != nil {
		return "", xerrors.Errorf("fail to download metaData: %s", err)
	}
	util.LogPrintf("Finish downloading metaFile")

	/* create lattice */
	// create getter
	chunkNum := len(metaData.DataCIDIndexMap)
	getter := ipfsconnector.CreateIPFSGetter(c.IPFSConnector, metaData.DataCIDIndexMap, metaData.ParityCIDs)
	if len(option.DataFilter) > 0 {
		getter.DataFilter = make(map[int]struct{}, len(option.DataFilter))
		for _, index := range option.DataFilter {
			getter.DataFilter[index] = struct{}{}
		}
	}

	// create lattice
	lattice := entangler.NewLattice(metaData.Alpha, metaData.S, metaData.P, chunkNum, getter, 2)
	lattice.Init()
	util.LogPrintf("Finish generating lattice")

	/* download & recover file from IPFS */
	data, repaired, errDownload := c.downloadAndRecover(lattice, metaData, option)
	if errDownload != nil {
		err = errDownload
		return
	}

	/* write to file in the given path */
	return writeFile(rootCID, path, data, repaired)
}

// dataReupload re-uploads the recovered data back to IPFS
func (c *Client) dataReupload(chunk []byte, cid string, allow bool) error {
	if !allow {
		return nil
	}

	uploadCID, err := c.AddRawData(chunk)
	if err != nil {
		return xerrors.Errorf("fail to upload the repaired chunk to IPFS: %s", err)
	}
	if uploadCID != cid {
		return xerrors.Errorf("incorrect CID of the repaired chunk. Expected: %s, Got: %s", cid, uploadCID)
	}
	return nil
}

// writeFile writes the recovered data to the file at the output path
func writeFile(rootCID string, path string, data []byte, repaired bool) (out string, err error) {
	if len(path) == 0 {
		out = rootCID
	} else {
		out = path
	}

	err = os.WriteFile(out, data, 0600)
	if err == nil {
		if repaired {
			util.LogPrintf("Finish downloading file (recovered)")
		} else {
			util.LogPrintf("Finish downloading file (no recovery)")
		}
	}
	return out, err
}
