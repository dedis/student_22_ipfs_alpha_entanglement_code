package test

import (
	"bytes"
	"ipfs-alpha-entanglement-code/cmd"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"math/rand"
	"testing"
)

var alpha, s, p = 3, 5, 5

type FileInfo struct {
	fileCID    string
	metaCID    string
	totalBlock int
}

var InfoMap = map[string]FileInfo{
	"20MB": {
		fileCID:    "QmY4ShSx49sYCCZxpqQWMcbjv2hv4yWWp2yprrk53DPqvT",
		metaCID:    "QmeZmAZ7TiFRT7kqoV4oynn1STKwo8sbiwRnLsM21eZMCc",
		totalBlock: 81,
	},
	"25MB": {
		fileCID:    "QmNkkcM5tFMqWxdrekyZoJnF5QxWKZnqYdJFBUj1jssRhb",
		metaCID:    "QmcnV4N1umtzBRk5fC6e8TYkTKhgFkwqxN6LjPZvTworwZ",
		totalBlock: 101,
	},
}

type TestResult struct {
	successCnt     int
	recoverRate    float32
	downloadParity uint
}

var recovery = func(fileinfo FileInfo, missingData map[int]struct{}, missingParity []map[int]struct{}) (result TestResult) {
	c, err := cmd.NewClient()
	if err != nil {
		panic(err)
	}

	err = c.InitIPFSConnector()
	if err != nil {
		panic(err)
	}

	// download metafile
	metaData, err := c.GetMetaData(fileinfo.metaCID)
	if err != nil {
		panic(err)
	}

	chunkNum := len(metaData.DataCIDIndexMap)
	// create getter
	getter := ipfsconnector.CreateIPFSGetter(c.IPFSConnector, metaData.DataCIDIndexMap, metaData.ParityCIDs)
	getter.DataFilter = missingData
	getter.ParityFilter = missingParity

	// create lattice
	lattice := entangler.NewLattice(metaData.Alpha, metaData.S, metaData.P, chunkNum, getter, 2)
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
			uploadCID, err := c.AddRawData(chunk)
			if err != nil {
				return
			}
			if uploadCID != cid {
				return
			}
		}
		successCount++

		// unmarshal and iterate
		dagNode, err := c.GetDagNodeFromRawBytes(chunk)
		if err != nil {
			return
		}
		links := dagNode.Links()
		for _, link := range links {
			walker(link.Cid.String())
		}
	}
	walker(fileinfo.fileCID)

	result.successCnt = successCount
	result.recoverRate = float32(successCount) / float32(fileinfo.totalBlock)

	var downloadParity uint = 0
	for _, parities := range lattice.ParityBlocks {
		for _, parity := range parities {
			if len(parity.Data) > 0 {
				downloadParity++
			}
		}
	}
	result.downloadParity = downloadParity

	return result
}

func Test_Only_Data_Loss(t *testing.T) {
	onlyData := func(missNum int, fileinfo FileInfo) func(*testing.T) {
		return func(*testing.T) {
			indexes := make([]int, fileinfo.totalBlock)
			for i := 0; i < fileinfo.totalBlock; i++ {
				indexes[i] = i
			}
			missedIndexes := map[int]struct{}{}
			for i := 0; i < missNum; i++ {
				r := int(rand.Int63n(int64(len(indexes))))
				missedIndexes[indexes[r]] = struct{}{}
				indexes[r], indexes[len(indexes)-1] = indexes[len(indexes)-1], indexes[r]
				indexes = indexes[:len(indexes)-1]
			}
			result := recovery(fileinfo, missedIndexes, nil)
			t.Logf("Data Recovery Rate: %f", result.recoverRate)
			t.Logf("Parity Overhead: %d", result.downloadParity)
			t.Logf("Successfully Downloaded Block: %d", result.successCnt)
		}
	}

	// missNum: 1
	// Success Data Recovery Rate: 1.000000
	// Success Parity Overhead: 2
	// missNum: 81 (All blocks are missing)
	// Success Data Recovery Rate: 1.000000
	// Success Parity Overhead: 81
	t.Run("test", onlyData(100, InfoMap["25MB"]))
}

func Test_Only_Parity_Loss(t *testing.T) {
	//var allRates []float32
	//var allOverhead []float32
	var accuRate float32
	var accuOverhead uint
	var accuSuccessCnt int

	onlyParity := func(missNum int, fileinfo FileInfo) func(*testing.T) {
		return func(*testing.T) {
			indexes := make([][]int, 3)
			for i := range indexes {
				indexes[i] = make([]int, fileinfo.totalBlock)
			}
			for i := range indexes {
				for j := 0; j < fileinfo.totalBlock; j++ {
					indexes[i][j] = j
				}
			}

			/* All data block is missing */
			missedDataIndexes := map[int]struct{}{}
			for i := 0; i < fileinfo.totalBlock; i++ {
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
			result := recovery(fileinfo, missedDataIndexes, missedParityIndexes)
			t.Logf("Data Recovery Rate: %f", result.recoverRate)
			t.Logf("Parity Overhead: %d", result.downloadParity)
			t.Logf("Successfully Downloaded Block: %d", result.successCnt)
			accuRate += result.recoverRate
			accuOverhead += result.downloadParity
			accuSuccessCnt += result.successCnt
		}
	}

	//for missingCnt := 0; missingCnt < 312; missingCnt++ {
	//
	//}
	t.Run("test", onlyParity(100, InfoMap["25MB"]))
}
