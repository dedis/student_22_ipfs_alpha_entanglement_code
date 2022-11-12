package ipfsconnector

import (
	"ipfs-alpha-entanglement-code/entangler"
	"sync"

	"golang.org/x/xerrors"
)

type IPFSGetter struct {
	entangler.BlockGetter

	*IPFSConnector
	indexCIDMap  SafeMap
	parityCIDMap []SafeMap
}

func CreateIPFSGetter(connector *IPFSConnector, numParity int) (getter *IPFSGetter) {
	parityMap := make([]SafeMap, numParity)
	for i := 0; i < numParity; i++ {
		parityMap[i] = SafeMap{&sync.RWMutex{}, map[int]string{}}
	}
	getter = &IPFSGetter{
		IPFSConnector: connector,
		indexCIDMap:   SafeMap{&sync.RWMutex{}, map[int]string{}},
		parityCIDMap:  parityMap,
	}
	return
}

func (getter *IPFSGetter) GetData(index int) (data []byte, err error) {
	// check if any cid is known
	cid, ok := getter.indexCIDMap.Get(index)
	if !ok {
		err = xerrors.Errorf("unable to find CID for index %d", index)
		return
	}

	// get the node
	node, err := getter.shell.ObjectGet(cid)
	if err != nil {
		return
	}
	for i, link := range node.Links {
		// TODO: add its children's CID to map
		childIdx := 0 + i
		getter.indexCIDMap.Add(childIdx, link.Hash)
	}
	data, err = getter.shell.BlockGet(cid)

	return
}

func (getter *IPFSGetter) GetParity(index int, strand int) (parity []byte, err error) {
	return
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
