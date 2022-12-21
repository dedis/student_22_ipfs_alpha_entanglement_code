package performance

import (
	"encoding/json"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"math/rand"

	"golang.org/x/xerrors"
)

// RepGetter handles connection to IPFS and mimics data & repetition loss
type RepGetter struct {
	*ipfsconnector.IPFSConnector

	DataIndexCIDMap util.SafeMap
	DataFilter      map[int]struct{}
	RepFilter       []map[int]struct{}

	cache map[string][]byte
}

func CreateRepGetter(connector *ipfsconnector.IPFSConnector, CIDIndexMap map[string]int) *RepGetter {
	indexToDataCIDMap := *util.NewSafeMap()
	indexToDataCIDMap.AddReverseMap(CIDIndexMap)
	return &RepGetter{
		IPFSConnector:   connector,
		DataIndexCIDMap: indexToDataCIDMap,
		cache:           map[string][]byte{},
	}
}

func (getter *RepGetter) GetData(index int) (data []byte, err error) {
	cid, ok := getter.DataIndexCIDMap.Get(index)
	if !ok {
		err := xerrors.Errorf("invalid index")
		return nil, err
	}

	if getter.DataFilter != nil {
		if _, ok = getter.DataFilter[index]; ok {
			/* get the rep, mask to represent the rep loss */
			if getter.RepFilter != nil {
				missing := true
				for _, repFilter := range getter.RepFilter {
					if repFilter == nil {
						continue
					}
					if _, ok := repFilter[index]; !ok {
						missing = false
						break
					}
				}
				if missing {
					err := xerrors.Errorf("no data exists")
					return nil, err
				}
			}
		}
	}

	// read from cache
	if data, ok := getter.cache[cid]; ok {
		return data, nil
	}
	// download from IPFS and store in cache
	data, err = getter.GetRawBlock(cid)
	if err != nil {
		return nil, err
	}
	getter.cache[cid] = data
	return data, nil
}

var RepRecover = func(fileinfo FileInfo,
	metaData Metadata, getter *RepGetter) (result PerfResult) {

	conn := getter.IPFSConnector

	successCount := 0
	var walker func(string)
	walker = func(cid string) {
		chunk, err := getter.GetData(metaData.DataCIDIndexMap[cid])
		if err != nil {
			return
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

	result.PartialSuccessCnt = successCount
	result.RecoverRate = float32(successCount) / float32(fileinfo.TotalBlock)
	return result
}

var RepRecoverWithFilter = func(fileinfo FileInfo, missNum int, repFactor int, iteration int) PerfResult {
	avgResult := PerfResult{}

	// create IPFS connector
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

	// create getter
	getter := CreateRepGetter(conn, metaData.DataCIDIndexMap)

	// generate random parity loss and repeat tests
	for b := 0; b < iteration; b++ {
		indexes := make([][]int, repFactor)
		for i := range indexes {
			indexes[i] = make([]int, fileinfo.TotalBlock)
			for j := 0; j < fileinfo.TotalBlock; j++ {
				indexes[i][j] = j + 1
			}
		}

		/* All data block is missing */
		missedDataIndexes := map[int]struct{}{}
		for i := 0; i < fileinfo.TotalBlock; i++ {
			missedDataIndexes[i+1] = struct{}{}
		}

		/* Some replication block is missing */
		missedRepIndexes := make([]map[int]struct{}, repFactor)
		for i := 0; i < repFactor; i++ {
			missedRepIndexes[i] = map[int]struct{}{}
		}

		indexRange := repFactor * fileinfo.TotalBlock
		missingIndex := make(map[int]bool)
		for i := 0; i < missNum; i++ {
			r := int(rand.Int63n(int64(indexRange)))
			for missingIndex[r] {
				r = int(rand.Int63n(int64(indexRange)))
			}
			missingIndex[r] = true
		}
		for key := range missingIndex {
			outerIndex := key / fileinfo.TotalBlock
			innerIndex := key%fileinfo.TotalBlock + 1
			missedRepIndexes[outerIndex][innerIndex] = struct{}{}
		}
		getter.DataFilter = missedDataIndexes
		getter.RepFilter = missedRepIndexes

		result := RepRecover(fileinfo, metaData, getter)
		avgResult.RecoverRate += result.RecoverRate
		avgResult.DownloadParity += result.DownloadParity
		avgResult.PartialSuccessCnt += result.PartialSuccessCnt
		if result.PartialSuccessCnt == fileinfo.TotalBlock {
			avgResult.FullSuccessCnt++
		}
	}
	avgResult.RecoverRate = avgResult.RecoverRate / float32(iteration)
	avgResult.DownloadParity = avgResult.DownloadParity / float32(iteration)
	avgResult.PartialSuccessCnt = avgResult.PartialSuccessCnt / iteration
	avgResult.FullSuccessCnt = avgResult.FullSuccessCnt / float32(iteration)
	return avgResult
}

func Perf_Replication(fileCase string, missPercent float32, repFactor int, iteration int) PerfResult {
	// check the validity of test case
	fileinfo, ok := InfoMap[fileCase]
	if !ok {
		return PerfResult{Err: xerrors.Errorf("invalid test case")}
	}

	missNum := int(float32(fileinfo.TotalBlock*repFactor) * missPercent)
	return RepRecoverWithFilter(fileinfo, missNum, repFactor, iteration)
}
