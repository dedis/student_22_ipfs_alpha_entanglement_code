package entangler

import (
	"ipfs-alpha-entanglement-code/util"
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
	Alpha       int // TODO: now only support alpha = 3 ???
	S           int
	P           int
	ChunkSize   int
	TotalChunks int

	OriginData [][]byte
	HParities  []*EntangledBlock
	RHParities []*EntangledBlock
	LHParities []*EntangledBlock

	cachedParityH   []*EntangledBlock
	cachedParityRH  []*EntangledBlock
	cachedParityLH  []*EntangledBlock
	rightMostBlocks []*EntangledBlock

	finished bool
}

// NewEntangler takes the entanglement paramters and the original data slice and creates an entangler
func NewEntangler(alpha int, s int, p int, chunkSize int, data *[][]byte) (entangler *Entangler) {
	entangler = &Entangler{Alpha: alpha, S: s, P: p, ChunkSize: chunkSize}
	entangler.OriginData = *data
	entangler.TotalChunks = len(*data)
	entangler.finished = false

	return
}

// GetEntanglement returns the entangled blocks in different strands
func (e *Entangler) GetEntanglement() (HParities, RHParities, LHParities []*EntangledBlock) {
	if !e.finished {
		// lazy entangle
		util.LogPrint("Start entangling\n")
		e.Entangle()
		e.finished = true
		util.LogPrint("Finish entangling\n")
	}
	HParities = e.HParities
	RHParities = e.RHParities
	LHParities = e.LHParities

	return
}

// Entangle generate the entangelement for the given arrray of blocks
func (e *Entangler) Entangle() {
	e.PrepareEntangle()

	// generate the lattice
	util.LogPrint("Start generating lattice\n")
	for i, block := range e.OriginData {
		e.EntangleSingleBlock(i+1, block)
	}
	util.LogPrint("Finish generating lattice\n")

	// wraps the lattice
	util.LogPrint("Start wrapping lattice\n")
	e.WrapLattice()
	util.LogPrint("Finish wrapping lattice\n")
}

// PrepareEntangle prepares the data structure that will be used for entanglement
func (e *Entangler) PrepareEntangle() {
	e.HParities = make([]*EntangledBlock, len(e.OriginData))
	e.RHParities = make([]*EntangledBlock, len(e.OriginData))
	e.LHParities = make([]*EntangledBlock, len(e.OriginData))

	cachedParityH := make([]*EntangledBlock, e.S)
	cachedParityRH := make([]*EntangledBlock, e.P)
	cachedParityLH := make([]*EntangledBlock, e.P)
	for i := 0; i < e.S; i++ {
		cachedParityH[i] = &EntangledBlock{
			LeftBlockIndex: 0, RightBlockIndex: 0,
			Data: make([]byte, e.ChunkSize), Strand: Horizontal}
	}
	for i := 0; i < e.P; i++ {
		cachedParityRH[i] = &EntangledBlock{
			LeftBlockIndex: 0, RightBlockIndex: 0,
			Data: make([]byte, e.ChunkSize), Strand: Right}
		cachedParityLH[i] = &EntangledBlock{
			LeftBlockIndex: 0, RightBlockIndex: 0,
			Data: make([]byte, e.ChunkSize), Strand: Left}
	}
	e.cachedParityH = cachedParityH
	e.cachedParityRH = cachedParityRH
	e.cachedParityLH = cachedParityLH

	e.rightMostBlocks = make([]*EntangledBlock, 0)
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
	e.cachedParityH[hCached] = hNext
	e.HParities[index-1] = hNext
	rNext := &EntangledBlock{
		LeftBlockIndex: index, RightBlockIndex: rIndex,
		Data: rParityData, Strand: Right}
	e.cachedParityRH[rCached] = rNext
	e.RHParities[index-1] = rNext
	lNext := &EntangledBlock{
		LeftBlockIndex: index, RightBlockIndex: lIndex,
		Data: lParityData, Strand: Left}
	e.cachedParityLH[lCached] = lNext
	e.LHParities[index-1] = lNext
}

func (e *Entangler) WrapLattice() {
	for _, parityNode := range e.cachedParityH {
		// Link the last parity block to the first data block of the chain
		index, _, _ := e.GetChainStartIndex(parityNode.RightBlockIndex)
		parityNode.RightBlockIndex = index
		// Recompute the first parity block
		hIndex, _, _ := e.GetForwardNeighborIndexes(index)
		if e.CheckValid(hIndex) {
			// the first block is not the rightmost block
			rNext := &EntangledBlock{
				LeftBlockIndex: index, RightBlockIndex: hIndex,
				Data: e.XORBlockData(e.OriginData[index-1], parityNode.Data), Strand: Horizontal}
			e.HParities[index-1] = rNext
		}
	}
	util.LogPrint("Finish wrapping horizontal strand\n")
	for _, parityNode := range e.cachedParityRH {
		// Link the last parity block to the first data block of the chain
		_, index, _ := e.GetChainStartIndex(parityNode.RightBlockIndex)
		parityNode.RightBlockIndex = index
		// fmt.Println(parityNode.LeftBlockIndex, parityNode.RightBlockIndex, r)
		// Recompute the first parity block
		_, rIndex, _ := e.GetForwardNeighborIndexes(index)
		if e.CheckValid(rIndex) {
			// the first block is not the rightmost block
			rNext := &EntangledBlock{
				LeftBlockIndex: index, RightBlockIndex: rIndex,
				Data: e.XORBlockData(e.OriginData[index-1], parityNode.Data), Strand: Right}
			e.RHParities[index-1] = rNext
		}
	}
	util.LogPrint("Finish wrapping right-hand strand\n")
	for _, parityNode := range e.cachedParityLH {
		// Link the last parity block to the first data block of the chain
		_, _, index := e.GetChainStartIndex(parityNode.RightBlockIndex)
		parityNode.RightBlockIndex = index
		// Recompute the first parity block
		_, _, lIndex := e.GetForwardNeighborIndexes(index)
		if e.CheckValid(lIndex) {
			// the first block is not the rightmost block
			rNext := &EntangledBlock{
				LeftBlockIndex: index, RightBlockIndex: lIndex,
				Data: e.XORBlockData(e.OriginData[index-1], parityNode.Data), Strand: Left}
			e.LHParities[index-1] = rNext
		}
	}
	util.LogPrint("Finish wrapping left-hand strand\n")
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

	rh = (y - x + e.P) % e.P
	lh = (x + y) % e.S

	return
}

// GetChainStartIndexes returns the position of the first node on the chain where the indexed node is on
func (e *Entangler) GetChainStartIndex(index int) (h, rh, lh int) {
	h, rh, lh = e.GetChainIndexes(index)
	h = h + 1
	rh = (e.P-rh)%e.P + 1
	lh = lh + 1

	return
}

// GetBackwardNeighborIndexes returns the index of backward neighbors that can be entangled with current node
func (e *Entangler) GetBackwardNeighborIndexes(index int) (h, rh, lh int) {
	if e.Alpha != 3 {
		util.ThrowError("alpha should equal 3")
	}

	// d_i is tangled with p_{h,i}
	pos := e.GetPositionCategory(index)
	switch pos {
	case Top:
		h = index - e.S
		rh = index - e.S*e.P + (e.S*e.S - 1)
		lh = index - (e.S - 1)
	case Central:
		h = index - e.S
		rh = index - (e.S + 1)
		lh = index - (e.S - 1)
	case Bottom:
		h = index - e.S
		rh = index - (e.S + 1)
		lh = index - e.S*e.P + (e.S-1)*(e.S-1)
	}

	return
}

// GetForwardNeighborIndexes returns the index of forward neighbors that is the entangled output of current node
func (e *Entangler) GetForwardNeighborIndexes(index int) (h, rh, lh int) {
	if e.Alpha != 3 {
		util.ThrowError("alpha should equal 3")
	}

	// d_i creates entangled block p_{i,j}
	pos := e.GetPositionCategory(index)
	switch pos {
	case Top:
		h = index + e.S
		rh = index + e.S + 1
		lh = index + e.S*e.P - (e.S-1)*(e.S-1)
	case Central:
		h = index + e.S
		rh = index + e.S + 1
		lh = index + e.S - 1
	case Bottom:
		h = index + e.S
		rh = index + e.S*e.P - (e.S*e.S - 1)
		lh = index + e.S - 1
	}

	return
}

// func (e *Entangler) GetRightmostBlocks() (HParities, RHParities, LHParities []*EntangledBlock) {
// 	HParities = make([]*EntangledBlock, 0)
// 	RHParities = make([]*EntangledBlock, 0)
// 	LHParities = make([]*EntangledBlock, 0)

// 	maxWapNum :=
// }

func (e *Entangler) CheckValid(index int) bool {
	if index < 1 || index > e.TotalChunks {
		return false
	}
	return true
}

// XORBlockData pads the bytes to the desired length and XOR these two bytes array
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
		util.ThrowError("data block size should not be larger than %d. Now is %d", e.ChunkSize, dataLength)
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
