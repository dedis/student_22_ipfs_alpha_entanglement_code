package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"math/rand"

	"golang.org/x/xerrors"
)

type RecoverGetter struct {
	entangler.BlockGetter
	*ipfsconnector.IPFSConnector
	DataIndexCIDMap util.SafeMap
	DataFilter      map[int]struct{}
	Parity          [][]string
	ParityFilter    []map[int]struct{}

	BlockNum int

	cache map[string][]byte
}

func CreateRecoverGetter(connector *ipfsconnector.IPFSConnector, CIDIndexMap map[string]int, parityCIDs [][]string) (*RecoverGetter, error) {
	indexToDataCIDMap := *util.NewSafeMap()
	indexToDataCIDMap.AddReverseMap(CIDIndexMap)
	getter := RecoverGetter{
		IPFSConnector:   connector,
		DataIndexCIDMap: indexToDataCIDMap,
		Parity:          parityCIDs,
		BlockNum:        len(CIDIndexMap),
		cache:           map[string][]byte{},
	}

	err := getter.InitCache()

	return &getter, err
}

func (getter *RecoverGetter) InitCache() error {
	// init data
	for _, dataCID := range getter.DataIndexCIDMap.GetAll() {
		// download from IPFS and store in cache
		data, err := getter.GetRawBlock(dataCID)
		if err != nil {
			return err
		}
		getter.cache[dataCID] = data
	}

	// init parities
	for _, parities := range getter.Parity {
		for _, parityCID := range parities {
			// download from IPFS and store in cache
			data, err := getter.GetFileToMem(parityCID)
			if err != nil {
				return err
			}
			getter.cache[parityCID] = data
		}
	}

	return nil
}

func (getter *RecoverGetter) GetData(index int) ([]byte, error) {
	/* Get the target CID of the block */
	cid, ok := getter.DataIndexCIDMap.Get(index)
	if !ok {
		err := xerrors.Errorf("invalid index")
		return nil, err
	}

	/* get the data, mask to represent the data loss */
	if getter.DataFilter != nil {
		if _, ok = getter.DataFilter[index]; ok {
			err := xerrors.Errorf("no data exists")
			return nil, err
		}
	}

	// read from cache
	if data, ok := getter.cache[cid]; ok {
		return data, nil
	}
	return nil, xerrors.Errorf("no such data")
}

func (getter *RecoverGetter) GetParity(index int, strand int) ([]byte, error) {
	if index < 1 || index > getter.BlockNum {
		err := xerrors.Errorf("invalid index")
		return nil, err
	}
	if strand < 0 || strand > len(getter.Parity) {
		err := xerrors.Errorf("invalid strand")
		return nil, err
	}

	/* Get the target CID of the block */
	cid := getter.Parity[strand][index-1]

	/* Get the parity, mask to represent the parity loss */
	if getter.ParityFilter != nil && len(getter.ParityFilter) > strand && getter.ParityFilter[strand] != nil {
		if _, ok := getter.ParityFilter[strand][index]; ok {
			err := xerrors.Errorf("no parity exists")
			return nil, err
		}
	}

	// read from cache
	if data, ok := getter.cache[cid]; ok {
		return data, nil
	}
	return nil, xerrors.Errorf("no such parity")
}

var Recovery = func(fileinfo FileInfo, metaData Metadata, getter *RecoverGetter) (result PerfResult) {
	conn := getter.IPFSConnector
	chunkNum := len(metaData.DataCIDIndexMap)

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
			// 	uploadCID, err := conn.AddRawData(chunk)
			// 	if err != nil {
			// 		return
			// 	}
			// 	if uploadCID != cid {
			// 		return
			// 	}
		}
		successCount++

		// unmarshal and iterate
		dagNode, err := conn.GetDagNodeFromRawBytes(chunk)
		if err != nil {
			fmt.Println(err)
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

	var downloadParity uint = 0
	for _, parities := range lattice.ParityBlocks {
		for _, parity := range parities {
			if len(parity.Data) > 0 {
				downloadParity++
			}
		}
	}
	result.DownloadParity = float32(downloadParity)

	return result
}

var RecoverWithFilter = func(fileinfo FileInfo, missNum int, iteration int, nbNodes int) (result PerfResult) {
	avgResult := PerfResult{}

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
	getter, err := CreateRecoverGetter(conn, metaData.DataCIDIndexMap, metaData.ParityCIDs)
	if err != nil {
		return PerfResult{Err: err}
	}

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
			missedDataIndexes[i+1] = struct{}{}
		}

		/* Some parity block is missing */
		missedParityIndexes := make([]map[int]struct{}, alpha)
		for i := 0; i < alpha; i++ {
			missedParityIndexes[i] = map[int]struct{}{}
		}
		if nbNodes == 0 {
			for i := 0; i < missNum; i++ {
				rOuter := int(rand.Int63n(int64(alpha)))
				for len(indexes[rOuter]) == 0 {
					rOuter = int(rand.Int63n(int64(alpha)))
				}
				rInner := int(rand.Int63n(int64(len(indexes[rOuter]))))
				missedParityIndexes[rOuter][indexes[rOuter][rInner]] = struct{}{}
				indexes[rOuter][rInner], indexes[rOuter][len(indexes[rOuter])-1] =
					indexes[rOuter][len(indexes[rOuter])-1], indexes[rOuter][rInner]
				indexes[rOuter] = indexes[rOuter][:len(indexes[rOuter])-1]
			}
		} else {
			curIndex := 0
			nodeIndexes := make([][]int, 10)
			for i := 0; i < fileinfo.TotalBlock; i++ {
				for j := 0; j < 3; j++ {
					nodeIndexes[curIndex] = append(nodeIndexes[curIndex], j*fileinfo.TotalBlock+i)
					curIndex = (curIndex + 1) % nbNodes
				}
			}

			missedNodes := map[int]bool{}
			for i := 0; i < missNum; i++ {
				idx := int(rand.Int63n(int64(nbNodes)))
				for missedNodes[idx] {
					idx = int(rand.Int63n(int64(nbNodes)))
				}
				missedNodes[idx] = true
			}

			for key := range missedNodes {
				for _, v := range nodeIndexes[key] {
					j := v / fileinfo.TotalBlock
					i := v % fileinfo.TotalBlock
					missedParityIndexes[j][i+1] = struct{}{}
				}
			}
		}
		getter.DataFilter = missedDataIndexes
		getter.ParityFilter = missedParityIndexes

		result := Recovery(fileinfo, metaData, getter)
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

func Perf_Recovery(fileCase string, missPercent float32, iteration int) PerfResult {
	fileinfo, ok := InfoMap[fileCase]
	if !ok {
		return PerfResult{Err: xerrors.Errorf("invalid test case")}
	}

	missNum := int(float32(fileinfo.TotalBlock*alpha) * missPercent)
	return RecoverWithFilter(fileinfo, missNum, iteration, 0)
}
