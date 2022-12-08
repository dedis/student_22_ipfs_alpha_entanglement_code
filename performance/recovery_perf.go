package performance

import (
	"bytes"
	"encoding/json"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"math/rand"

	"golang.org/x/xerrors"
)

type PerfResult struct {
	SuccessCnt     int
	RecoverRate    float32
	DownloadParity uint
	Err            error
}

var Recovery = func(fileinfo FileInfo, missingData map[int]struct{}, missingParity []map[int]struct{}) (result PerfResult) {
	conn, err := ipfsconnector.CreateIPFSConnector(0)
	if err != nil {
		return PerfResult{Err: err}
	}

	// download metafile
	data, err := conn.GetFileToMem(fileinfo.MetaCID)
	if err != nil {
		return PerfResult{Err: err}
	}
	var metaData Metadata
	err = json.Unmarshal(data, &metaData)
	if err != nil {
		return PerfResult{Err: err}
	}

	chunkNum := len(metaData.DataCIDIndexMap)
	// create getter
	getter := ipfsconnector.CreateIPFSGetter(conn, metaData.DataCIDIndexMap, metaData.ParityCIDs)
	getter.DataFilter = missingData
	getter.ParityFilter = missingParity

	// create lattice
	lattice := entangler.NewLattice(metaData.Alpha, metaData.S, metaData.P, chunkNum, getter, 1)
	lattice.Init()

	// download & recover file from IPFS
	successCount := 0
	var walker func(string)
	walker = func(cid string) {
		chunk, hasRepaired, err := lattice.GetChunk(metaData.DataCIDIndexMap[cid])
		if err != nil {
			return
		}

		// upload missing chunk back to the network if allowed
		if hasRepaired {
			// TODO: does trimming zero always works?
			chunk = bytes.Trim(chunk, "\x00")
			uploadCID, err := conn.AddRawData(chunk)
			if err != nil {
				return
			}
			if uploadCID != cid {
				return
			}
		}
		successCount++

		// unmarshal and iterate
		dagNode, err := conn.GetDagNodeFromRawBytes(chunk)
		if err != nil {
			return
		}
		links := dagNode.Links()
		for _, link := range links {
			walker(link.Cid.String())
		}
	}
	walker(fileinfo.FileCID)

	result.SuccessCnt = successCount
	result.RecoverRate = float32(successCount) / float32(fileinfo.TotalBlock)

	var downloadParity uint = 0
	for _, parities := range lattice.ParityBlocks {
		for _, parity := range parities {
			if len(parity.Data) > 0 {
				downloadParity++
			}
		}
	}
	result.DownloadParity = downloadParity

	return result
}

func Perf_Recovery(fileCase string, missPercent float32, iteration int) PerfResult {
	fileinfo, ok := InfoMap[fileCase]
	if !ok {
		return PerfResult{Err: xerrors.Errorf("invalid test case")}
	}

	missNum := int(float32(fileinfo.TotalBlock) * missPercent)
	avgResult := PerfResult{}
	for i := 0; i < iteration; i++ {
		indexes := make([][]int, alpha)
		for i := range indexes {
			indexes[i] = make([]int, fileinfo.TotalBlock)
			for j := 0; j < fileinfo.TotalBlock; j++ {
				indexes[i][j] = j + 1
			}
		}

		/* All data block is missing */
		missedDataIndexes := map[int]struct{}{}
		for i := 0; i < fileinfo.TotalBlock; i++ {
			missedDataIndexes[i] = struct{}{}
		}

		/* Some parity block is missing */
		missedParityIndexes := []map[int]struct{}{{}, {}, {}}
		for i := 0; i < missNum; i++ {
			rOuter := int(rand.Int63n(int64(3)))
			for len(indexes[rOuter]) == 0 {
				rOuter = int(rand.Int63n(int64(3)))
			}
			rInner := int(rand.Int63n(int64(len(indexes[rOuter]))))
			missedParityIndexes[rOuter][indexes[rOuter][rInner]] = struct{}{}
			indexes[rOuter][rInner], indexes[rOuter][len(indexes[rOuter])-1] =
				indexes[rOuter][len(indexes[rOuter])-1], indexes[rOuter][rInner]
			indexes[rOuter] = indexes[rOuter][:len(indexes[rOuter])-1]
		}

		result := Recovery(fileinfo, missedDataIndexes, missedParityIndexes)
		avgResult.RecoverRate += result.RecoverRate
		avgResult.DownloadParity += result.DownloadParity
		avgResult.SuccessCnt += result.SuccessCnt
	}
	avgResult.RecoverRate = avgResult.RecoverRate / float32(iteration)
	avgResult.DownloadParity = avgResult.DownloadParity / uint(iteration)
	avgResult.SuccessCnt = avgResult.SuccessCnt / iteration

	return avgResult
}
