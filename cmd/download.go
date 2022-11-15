package cmd

import (
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"os"
)

// Download download the original file, repair it if metadata is provided
func (c *Client) Download(rootCID string, path string, allowUpload bool) (err error) {
	conn, err := ipfsconnector.CreateIPFSConnector(0)
	util.CheckError(err, "failed to spawn peer node")

	metaData, ok := c.GetMetaData(rootCID)
	if !ok {
		err = conn.GetFile(rootCID, path)
		if err == nil {
			util.LogPrint(util.Green("Finish downloading file (no repair)"))
		}

		return
	}

	chunkNum := len(metaData.CIDIndexMap)
	// create getter
	getter := ipfsconnector.IPFSGetter{}
	// create lattice
	lattice := entangler.NewLattice(metaData.Alpha, metaData.S, metaData.P, chunkNum, &getter)
	lattice.Init()
	util.LogPrint(util.Green("Finish generating lattice"))

	// download & recover file from IPFS
	data := []byte{}
	cid := rootCID
	repaired := false
	for i := 0; i < chunkNum; i++ {
		// pre-order traversal
		chunk, hasRepaired, err := lattice.GetChunk(metaData.CIDIndexMap[cid])
		util.CheckError(err, "Fail to recover chunk with CID: %s", cid)

		repaired = repaired || hasRepaired
		if allowUpload && hasRepaired {
			// upload missing chunk back to the network
			uploadCID, err := conn.AddRawData(chunk)
			if err != nil || uploadCID != cid {
				util.ThrowError("fail to upload the repaired chunk to IPFS")
			}
		}
		data = append(data, chunk...)
	}

	// write to file in the given path
	err = os.WriteFile(path, data, 0644)
	if err == nil {
		if repaired {
			util.LogPrint(util.Green("Finish downloading file (repair)"))
		} else {
			util.LogPrint(util.Green("Finish downloading file (no repair)"))
		}
	}

	return
}
