package ipfsconnector

import (
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"

	"golang.org/x/xerrors"
)

type IPFSGetter struct {
	entangler.BlockGetter
	*IPFSConnector
	DataIndexCIDMap util.SafeMap
	DataFilter      map[int]struct{}
	Parity          [][]string
	ParityFilter    []map[int]struct{}

	BlockNum int
}

func CreateIPFSGetter(connector *IPFSConnector, CIDIndexMap map[string]int, parityCIDs [][]string) *IPFSGetter {
	indexToDataCIDMap := *util.NewSafeMap()
	indexToDataCIDMap.AddReverseMap(CIDIndexMap)
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
	if getter.DataFilter != nil {
		if _, ok = getter.DataFilter[index]; ok {
			err := xerrors.Errorf("no data exists")
			return nil, err
		}
	}
	data, err := getter.GetRawBlock(cid)
	return data, err

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
	if getter.ParityFilter != nil && len(getter.ParityFilter) > strand && getter.ParityFilter[strand] != nil {
		if _, ok := getter.ParityFilter[strand][index]; ok {
			err := xerrors.Errorf("no parity exists")
			return nil, err
		}
	}

	data, err := getter.GetFileToMem(cid)
	return data, err

}
