package ipfsconnector

import (
	"ipfs-alpha-entanglement-code/entangler"
	"sync"

	"golang.org/x/xerrors"
)

type IPFSGetter struct {
	entangler.BlockGetter
	*IPFSConnector
	DataIndexCIDMap SafeMap
	DataFilter      map[int]struct{}
	Parity          [][]string
	ParityFilter    map[int]struct{}

	BlockNum int
}

func CreateIPFSGetter(connector *IPFSConnector, CIDIndexMap map[string]int, parityCIDs [][]string) *IPFSGetter {
	indexToDataCIDMap := SafeMap{&sync.RWMutex{}, map[int]string{}}
	for cid, index := range CIDIndexMap {
		indexToDataCIDMap.Add(index, cid)
	}
	return &IPFSGetter{
		IPFSConnector:   connector,
		DataIndexCIDMap: indexToDataCIDMap,
		Parity:          parityCIDs,
		BlockNum:        len(CIDIndexMap),
	}
}

func (getter *IPFSGetter) GetData(index int) ([]byte, error) {
	/* Get the target CID of the block */
	cid, ok := getter.DataIndexCIDMap.Get(index)
	if !ok {
		err := xerrors.Errorf("invalid index")
		return nil, err
	}

	/* get the data, mask to represent the data loss */
	if _, ok = getter.DataFilter[index]; ok {
		err := xerrors.Errorf("no data exists")
		return nil, err
	} else {
		data, err := getter.GetRawBlock(cid)
		return data, err
	}
}

func (getter *IPFSGetter) GetParity(index int, strand int) ([]byte, error) {
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
	if _, ok := getter.ParityFilter[index]; ok {
		err := xerrors.Errorf("no parity exists")
		return nil, err
	} else {
		data, err := getter.GetFileToMem(cid)
		return data, err
	}
}

type SafeMap struct {
	*sync.RWMutex
	unsafeMap map[int]string
}

func (m *SafeMap) Add(key int, val string) {
	m.Lock()
	defer m.Unlock()

	m.unsafeMap[key] = val
}

func (m *SafeMap) Get(key int) (val string, ok bool) {
	m.RLock()
	defer m.RUnlock()

	val, ok = m.unsafeMap[key]
	return
}
