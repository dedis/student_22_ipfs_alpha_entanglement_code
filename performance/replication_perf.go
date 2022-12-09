package performance

import (
	"encoding/json"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"math/rand"

	"golang.org/x/xerrors"
)

type RepGetter struct {
	*ipfsconnector.IPFSConnector

	DataIndexCIDMap util.SafeMap
	DataFilter      map[int]struct{}
	RepFilter       []map[int]struct{}
}

func CreateRepGetter(connector *ipfsconnector.IPFSConnector, CIDIndexMap map[string]int) *RepGetter {
	indexToDataCIDMap := *util.NewSafeMap()
	indexToDataCIDMap.AddReverseMap(CIDIndexMap)
	return &RepGetter{
		IPFSConnector:   connector,
		DataIndexCIDMap: indexToDataCIDMap,
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
					if repFilter != nil {
						missing = false
						break
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
	data, err = getter.GetRawBlock(cid)
	return data, err
}

var RepRecover = func(fileinfo FileInfo, missingData map[int]struct{}, missingReplication []map[int]struct{}) (result PerfResult) {
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

	getter := CreateRepGetter(conn, metaData.DataCIDIndexMap)
	getter.DataFilter = missingData
	getter.RepFilter = missingReplication

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

	result.SuccessCnt = successCount
	result.RecoverRate = float32(successCount) / float32(fileinfo.TotalBlock)
	return result
}

func Perf_Replication(fileCase string, missPercent float32, repFactor int, iteration int) PerfResult {
	fileinfo, ok := InfoMap[fileCase]
	if !ok {
		return PerfResult{Err: xerrors.Errorf("invalid test case")}
	}

	missNum := int(float32(fileinfo.TotalBlock) * missPercent)
	avgResult := PerfResult{}
	for i := 0; i < iteration; i++ {
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
			missedDataIndexes[i] = struct{}{}
		}

		/* Some replication block is missing */
		missedRepIndexes := make([]map[int]struct{}, repFactor)
		for i := 0; i < repFactor; i++ {
			missedRepIndexes[i] = map[int]struct{}{}
		}
		for i := 0; i < missNum; i++ {
			rOuter := int(rand.Int63n(int64(repFactor)))
			for len(indexes[rOuter]) == 0 {
				rOuter = int(rand.Int63n(int64(repFactor)))
			}
			rInner := int(rand.Int63n(int64(len(indexes[rOuter]))))
			missedRepIndexes[rOuter][indexes[rOuter][rInner]] = struct{}{}
			indexes[rOuter][rInner], indexes[rOuter][len(indexes[rOuter])-1] =
				indexes[rOuter][len(indexes[rOuter])-1], indexes[rOuter][rInner]
			indexes[rOuter] = indexes[rOuter][:len(indexes[rOuter])-1]
		}

		result := RepRecover(fileinfo, missedDataIndexes, missedRepIndexes)
		avgResult.RecoverRate += result.RecoverRate
		avgResult.DownloadParity += result.DownloadParity
		avgResult.SuccessCnt += result.SuccessCnt
	}
	avgResult.RecoverRate = avgResult.RecoverRate / float32(iteration)
	avgResult.DownloadParity = avgResult.DownloadParity / uint(iteration)
	avgResult.SuccessCnt = avgResult.SuccessCnt / iteration

	return avgResult
}
