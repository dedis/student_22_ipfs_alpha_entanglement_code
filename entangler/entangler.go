package entangler

import (
	"fmt"
)

// Position defines which position category the original block belongs
type PositionClass int8

const (
	Top PositionClass = iota
	Central
	Bottom
)

// Strand defines which strand the entangled block belongs
type StrandClass int

const (
	Horizontal StrandClass = iota
	Right
	Left
)

// EntangledBlock is the parity block output by alpha entanglement code
type EntangledBlock struct {
	LeftBlockIndex  int
	RightBlockIndex int
	Data            []byte
	Strand          StrandClass
}

// Entangler manages all the entanglement related behaviors
type Entangler struct {
	Alpha     int // TODO: now only support alpha = 3 ???
	S         int
	P         int
	ChunkSize int

	OriginData [][]byte
	HParities  []*EntangledBlock
	RHParities []*EntangledBlock
	LHParities []*EntangledBlock

	cachedParityH  []*EntangledBlock
	cachedParityRH []*EntangledBlock
	cachedParityLH []*EntangledBlock
}

func NewEntangler(alpha int, s int, p int, chunkSize int, data *[][]byte) (entangler *Entangler) {
	entangler = &Entangler{Alpha: alpha, S: s, P: p, ChunkSize: chunkSize}

	entangler.OriginData = *data
	entangler.HParities = make([]*EntangledBlock, len(*data))
	entangler.RHParities = make([]*EntangledBlock, len(*data))
	entangler.LHParities = make([]*EntangledBlock, len(*data))

	entangler.cachedParityH = make([]*EntangledBlock, s)
	entangler.cachedParityRH = make([]*EntangledBlock, p)
	entangler.cachedParityLH = make([]*EntangledBlock, p)

	return
}

// GetEntanglement returns the entangled blocks in different strands
func (e *Entangler) GetEntanglement() (HParities, RHParities, LHParities []*EntangledBlock) {
	if len(e.HParities) == 0 || len(e.RHParities) == 0 || len(e.LHParities) == 0 {
		// lazy entangle
		e.Entangle()
	}
	HParities = e.HParities
	RHParities = e.RHParities
	LHParities = e.LHParities

	return
}

// Entangle generate the entangelement for the given arrray of blocks
func (e *Entangler) Entangle() {
	// generate the lattice
	for i, block := range e.OriginData {
		e.EntangleSingleBlock(i+1, block)
	}

	// wraps the lattice
	e.WrapLattice()
}

// EntangleSingleBlock reads the backward parity neighbors from cache and produce the corresponding forward parity neighbors
// It should be called in the correct order to ensure the correctness of cached blocks
func (e *Entangler) EntangleSingleBlock(index int, data []byte) {
	// read parity block from cache
	hCached, rCached, lCached := e.GetChainIndexes(index)
	hPrev := e.cachedParityH[hCached]
	rPrev := e.cachedParityRH[rCached]
	lPrev := e.cachedParityLH[lCached]

	// generate new parity block
	hParityData := e.XORBlockData(data, hPrev.Data)
	rParityData := e.XORBlockData(data, rPrev.Data)
	lParityData := e.XORBlockData(data, lPrev.Data)

	// generate, cache and store entangled block
	hIndex, rIndex, lIndex := e.GetForwardNeighborIndexes(index)
	hNext := &EntangledBlock{
		LeftBlockIndex: index, RightBlockIndex: hIndex,
		Data: hParityData, Strand: Horizontal}
	e.cachedParityRH[hCached] = hNext
	e.HParities[index-1] = hNext
	rNext := &EntangledBlock{
		LeftBlockIndex: index, RightBlockIndex: rIndex,
		Data: rParityData, Strand: Right}
	e.cachedParityRH[rCached] = rNext
	e.RHParities[index-1] = rNext
	lNext := &EntangledBlock{
		LeftBlockIndex: index, RightBlockIndex: lIndex,
		Data: lParityData, Strand: Left}
	e.cachedParityRH[lCached] = lNext
	e.LHParities[index-1] = lNext
}

func (e *Entangler) WrapLattice() {
	for _, parityNode := range e.cachedParityH {
		// Link the last parity block to the first data block of the chain
		h, _, _ := e.GetChainStartPosition(parityNode.RightBlockIndex)
		index := h + 1
		parityNode.LeftBlockIndex = index
		// Recompute the first parity block
		hIndex, _, _ := e.GetForwardNeighborIndexes(index)
		rNext := &EntangledBlock{
			LeftBlockIndex: index, RightBlockIndex: hIndex,
			Data: e.XORBlockData(e.OriginData[h], parityNode.Data), Strand: Horizontal}
		e.HParities[h] = rNext
	}
	for _, parityNode := range e.cachedParityRH {
		// Link the last parity block to the first data block of the chain
		_, r, _ := e.GetChainStartPosition(parityNode.RightBlockIndex)
		index := r + 1
		parityNode.LeftBlockIndex = index
		// Recompute the first parity block
		_, rIndex, _ := e.GetForwardNeighborIndexes(index)
		rNext := &EntangledBlock{
			LeftBlockIndex: index, RightBlockIndex: rIndex,
			Data: e.XORBlockData(e.OriginData[r], parityNode.Data), Strand: Horizontal}
		e.HParities[r] = rNext
	}
	for _, parityNode := range e.cachedParityH {
		// Link the last parity block to the first data block of the chain
		_, _, l := e.GetChainStartPosition(parityNode.RightBlockIndex)
		index := l + 1
		parityNode.LeftBlockIndex = index
		// Recompute the first parity block
		_, _, lIndex := e.GetForwardNeighborIndexes(index)
		rNext := &EntangledBlock{
			LeftBlockIndex: index, RightBlockIndex: lIndex,
			Data: e.XORBlockData(e.OriginData[l], parityNode.Data), Strand: Horizontal}
		e.HParities[l] = rNext
	}
}

// GetPositionCategory determines which category the node belongs. Top, Bottom or Central
func (e *Entangler) GetPositionCategory(index int) PositionClass {
	nodePos := index % e.S
	if nodePos == 1 || nodePos == 1-e.S {
		return Top
	} else if nodePos == 0 {
		return Bottom
	}
	return Central
}

// GetChainIndexes reads the cached backward parity neighbors of the current indexed node
func (e *Entangler) GetChainIndexes(index int) (h, rh, lh int) {
	h = (index - 1) % e.S

	indexInWindow := (index - 1) % (e.S * e.P)
	x := indexInWindow % e.P
	y := indexInWindow / e.P

	rh = (x - y + e.P) % e.P
	lh = (x + y) % e.S

	return
}

// GetChainStartIndexes returns the position of the first node on the chain where the indexed node is on
func (e *Entangler) GetChainStartPosition(index int) (h, rh, lh int) {
	// TODO: Check if the first node index is calculated correctly
	return e.GetChainIndexes(index)
}

// GetBackwardNeighborIndexes returns the index of backward neighbors that can be entangled with current node
func (e *Entangler) GetBackwardNeighborIndexes(index int) (h, rh, lh int) {
	if e.Alpha != 3 {
		panic(fmt.Errorf("alpha should equal 3"))
	}

	// d_i is tangled with p_{h,i}
	pos := e.GetPositionCategory(index)
	switch pos {
	case Top:
		h = index - e.S
		rh = index - e.S*e.P + (e.S*e.S - 1)
		lh = index - (e.S - 1)
	case Bottom:
		h = index - e.S
		rh = index - (e.S + 1)
		lh = index - (e.S - 1)
	case Central:
		h = index - e.S
		rh = index - (e.S + 1)
		lh = index - e.S*e.P + (e.S-1)*(e.S-1)
	}

	return
}

// GetForwardNeighborIndexes returns the index of forward neighbors that is the entangled output of current node
func (e *Entangler) GetForwardNeighborIndexes(index int) (h, rh, lh int) {
	if e.Alpha != 3 {
		panic(fmt.Errorf("alpha should equal 3"))
	}

	// d_i creates entangled block p_{i,j}
	pos := e.GetPositionCategory(index)
	switch pos {
	case Top:
		h = index + e.S
		rh = index + e.S + 1
		lh = index + e.S*e.P - (e.S-1)*(e.S-1)
	case Bottom:
		h = index + e.S
		rh = index + e.S + 1
		lh = index + e.S - 1
	case Central:
		h = index + e.S
		rh = index + e.S*e.P - (e.S*e.S - 1)
		lh = index + e.S - 1
	}

	return
}

func (e *Entangler) XORBlockData(data1 []byte, data2 []byte) (result []byte) {
	if len(data1) == 0 {
		return e.PaddedData(&data2)
	}
	if len(data2) == 0 {
		return e.PaddedData(&data1)
	}

	padded1 := e.PaddedData(&data1)
	padded2 := e.PaddedData(&data2)

	result = make([]byte, e.ChunkSize)
	for i := 0; i < e.ChunkSize; i++ {
		result[i] = padded1[i] ^ padded2[i]
	}

	return
}

// PaddedData pads the data to fixed length and return the padded data
func (e *Entangler) PaddedData(data *[]byte) (result []byte) {
	dataLength := len(*data)
	if dataLength > e.ChunkSize {
		panic(fmt.Errorf("data block size should not be larger than %d. Now is %d", e.ChunkSize, dataLength))
	}

	if dataLength < e.ChunkSize {
		// TODO: Decide another way of padding? Should origin length be recorded?
		result = make([]byte, e.ChunkSize)
		copy(result, *data)
	} else {
		result = *data
	}

	return
}
