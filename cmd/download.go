package cmd

import (
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"os"
	"strconv"
)

// Download download the original file, repair it if metadata is provided
func (c *Client) Download(rootCID string, path string, allowUpload bool) (err error) {
	conn, err := ipfsconnector.CreateIPFSConnector(0)
	util.CheckError(err, "failed to connect to IPFS node")

	metaData, ok := c.GetMetaData(rootCID)
	if !ok {
		err = conn.GetFile(rootCID, path)
		if err == nil {
			util.LogPrint("Finish downloading file (no repair)")
		}

		return
	}

	chunkNum := len(metaData.DataCIDIndexMap)
	// create getter
	getter := ipfsconnector.CreateIPFSGetter(conn, metaData.DataCIDIndexMap, metaData.ParityCIDs)
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
		util.CheckError(err, "fail to recover chunk with CID: %s", cid)
		// upload missing chunk back to the network if allowed
		repaired = repaired || hasRepaired
		if allowUpload && hasRepaired {
			uploadCID, err := conn.AddRawData(chunk)
			if err != nil || uploadCID != cid {
				util.ThrowError("fail to upload the repaired chunk to IPFS")
			}
		}

		links, err := conn.GetLinksFromRawBlock(chunk)
		util.CheckError(err, "fail to parse raw data")
		if len(links) > 0 {
			for _, link := range links {
				err = walker(link)
				if err != nil {
					return
				}
			}
		} else {
			data = append(data, chunk...)
			os.WriteFile(strconv.Itoa(len(data)), chunk, 0644)
		}
		return
	}
	walker(rootCID)

	// write to file in the given path
	err = os.WriteFile(path, data, 0644)
	if err == nil {
		if repaired {
			util.LogPrint("Finish downloading file (repair)")
		} else {
			util.LogPrint("Finish downloading file (no repair)")
		}
	}

	return
}
