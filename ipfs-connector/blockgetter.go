package ipfsconnector

import (
	"ipfs-alpha-entanglement-code/entangler"
	"sync"

	"golang.org/x/xerrors"
)

type IPFSGetter struct {
	entangler.BlockGetter
	*IPFSConnector
	Data            []string
	DataFilter      map[int]struct{}
	Parity          [][]string
	ParityFilter    map[int]struct{}
	indexToIndexMap SafeMap
}

func CreateIPFSGetter(connector *IPFSConnector, dataCIDs []string, parityCIDs [][]string, nodes []*TreeNode) *IPFSGetter {
	indexToIndexMap := SafeMap{&sync.RWMutex{}, map[int]int{}}
	for idx, node := range nodes {
		indexToIndexMap.Add(node.PreOrderIdx, idx)
	}
	return &IPFSGetter{
		IPFSConnector:   connector,
		Data:            dataCIDs,
		Parity:          parityCIDs,
		indexToIndexMap: indexToIndexMap,
	}
}

func (getter *IPFSGetter) GetData(index int) ([]byte, error) {
	if index < 1 || index > len(getter.Data) {
		err := xerrors.Errorf("invalid index")
		return nil, err
	}

	index, _ = getter.indexToIndexMap.Get(index - 1)
	/* Get the target CID of the block */
	cid := getter.Data[index]

	/* get the data, mask to represent the data loss */
	if _, ok := getter.DataFilter[index]; ok {
		err := xerrors.Errorf("no data exists")
		return nil, err
	} else {
		data, err := getter.shell.BlockGet(cid)
		return data, err
	}
}

func (getter *IPFSGetter) GetParity(index int, strand int) ([]byte, error) {
	if index < 1 || index > len(getter.Data) {
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
	if _, ok := getter.ParityFilter[index-1]; ok {
		err := xerrors.Errorf("no parity exists")
		return nil, err
	} else {
		data, err := getter.shell.BlockGet(cid)
		return data, err
	}
}

type SafeMap struct {
	*sync.RWMutex
	unsafeMap map[int]int
}

func (m *SafeMap) Add(key int, val int) {
	m.Lock()
	defer m.Unlock()

	m.unsafeMap[key] = val
}

func (m *SafeMap) Get(key int) (val int, ok bool) {
	m.RLock()
	defer m.RUnlock()

	val, ok = m.unsafeMap[key]
	return
}
