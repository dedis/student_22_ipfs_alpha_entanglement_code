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

// NewEntangledBlock creates a new entangled block and set it strandclass according to the input
func NewEntangledBlock(l int, r int, data []byte, strand int) (block *EntangledBlock) {
	block = &EntangledBlock{LeftBlockIndex: l, RightBlockIndex: r, Data: data}
	switch strand {
	case 0:
		block.Strand = Horizontal
	case 1:
		block.Strand = Right
	case 2:
		block.Strand = Left
	}

	return
}

// Entangler manages all the entanglement related behaviors
type Entangler struct {
	Alpha       int // TODO: now only support alpha = 3 ???
	S           int
	P           int
	ChunkSize   int
	TotalChunks int

	OriginData   [][]byte
	ParityBlocks [][]*EntangledBlock

	cachedParities  [][]*EntangledBlock
	rightMostBlocks []*EntangledBlock

	finished bool
}

// NewEntangler takes the entanglement paramters and the original data slice and creates an entangler
func NewEntangler(alpha int, s int, p int, chunkSize int, data *[][]byte) (entangler *Entangler) {
	if alpha == 1 {
		if s != 1 || p != 0 {
			util.ThrowError("invalid value. Expect s = 1 and p = 0")
		}
	} else if alpha > 1 {
		if s > p {
			util.ThrowError("invalid value. Expect p >= s")
		}
	} else {
		util.ThrowError("invalid value. Expect alpha > 0")
	}
	entangler = &Entangler{Alpha: alpha, S: s, P: p, ChunkSize: chunkSize}
	entangler.OriginData = *data
	entangler.TotalChunks = len(*data)
	entangler.finished = false

	return
}

// GetEntanglement returns the entanglement in bytes in different strands
func (e *Entangler) GetEntanglement() (entanglement [][]byte) {
	if !e.finished {
		// lazy entangle
		e.Entangle()
		e.finished = true
	}

	entanglement = make([][]byte, e.Alpha)
	for k := 0; k < e.Alpha; k++ {
		entangledData := make([]byte, 0)
		parities := e.ParityBlocks[k]
		util.LogPrint(util.Yellow("Strand %d: "), k)
		for _, parity := range parities {
			util.LogPrint(util.Yellow("(%d, %d) "), parity.LeftBlockIndex, parity.RightBlockIndex)
			entangledData = append(entangledData, parity.Data...)
		}
		util.LogPrint("\n")
		entanglement[k] = entangledData
	}

	return
}

// Entangle generate the entangelement for the given arrray of blocks
func (e *Entangler) Entangle() {
	e.PrepareEntangle()

	// generate the lattice
	util.LogPrint(util.White("Start generating lattice\n"))
	for i, block := range e.OriginData {
		e.EntangleSingleBlock(i+1, block)
	}
	util.LogPrint(util.White("Finish generating lattice\n"))

	// wraps the lattice
	util.LogPrint(util.White("Start wrapping lattice\n"))
	e.WrapLattice()
	util.LogPrint(util.White("Finish wrapping lattice\n"))
}

// PrepareEntangle prepares the data structure that will be used for entanglement
func (e *Entangler) PrepareEntangle() {
	e.ParityBlocks = make([][]*EntangledBlock, e.Alpha)
	for k := 0; k < e.Alpha; k++ {
		e.ParityBlocks[k] = make([]*EntangledBlock, e.TotalChunks)
	}

	e.cachedParities = make([][]*EntangledBlock, e.Alpha)
	e.cachedParities[0] = make([]*EntangledBlock, e.S)
	for i := 0; i < e.S; i++ {
		e.cachedParities[0][i] = NewEntangledBlock(0, 0, make([]byte, e.ChunkSize), 0)
	}
	for k := 1; k < e.Alpha; k++ {
		e.cachedParities[k] = make([]*EntangledBlock, e.P)
		for i := 0; i < e.P; i++ {
			e.cachedParities[k][i] = NewEntangledBlock(0, 0, make([]byte, e.ChunkSize), k)
		}
	}

	e.rightMostBlocks = make([]*EntangledBlock, 0)
}

// EntangleSingleBlock reads the backward parity neighbors from cache and produce the corresponding forward parity neighbors
// It should be called in the correct order to ensure the correctness of cached blocks
func (e *Entangler) EntangleSingleBlock(index int, data []byte) {
	cachePos := e.GetChainIndexes(index)
	rIndexes := e.GetForwardNeighborIndexes(index)

	for k := 0; k < e.Alpha; k++ {
		// read parity block from cache
		prevBlock := e.cachedParities[k][cachePos[k]]
		// generate new parity block
		parityData := e.XORBlockData(data, prevBlock.Data)
		// generate, cache and store entangled block
		nextBlock := NewEntangledBlock(index, rIndexes[k], parityData, k)
		e.cachedParities[k][cachePos[k]] = nextBlock
		e.ParityBlocks[k][index-1] = nextBlock
	}
}

func (e *Entangler) WrapLattice() {
	for k, cacheParity := range e.cachedParities {
		for _, parityNode := range cacheParity {
			// Link the last parity block to the first data block of the chain
			index := e.GetChainStartIndexes(parityNode.RightBlockIndex)[k]
			parityNode.RightBlockIndex = index
			// Recompute the first parity block
			rIndex := e.GetForwardNeighborIndexes(index)[k]
			if e.CheckValid(rIndex) {
				// the first block is not the rightmost block
				rNext := NewEntangledBlock(index, rIndex,
					e.XORBlockData(e.OriginData[index-1], parityNode.Data), k)
				e.ParityBlocks[k][index-1] = rNext
			}
		}
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
func (e *Entangler) GetChainIndexes(index int) (indexes []int) {
	h := (index - 1) % e.S

	indexInWindow := (index - 1) % (e.S * e.P)
	x := indexInWindow % e.P
	y := indexInWindow / e.P

	rh := (y - x + e.P) % e.P
	lh := (x + y) % e.S

	indexes = []int{h, rh, lh}

	return
}

// GetChainStartIndexes returns the position of the first node on the chain where the indexed node is on
func (e *Entangler) GetChainStartIndexes(index int) (indexes []int) {
	indexes = e.GetChainIndexes(index)
	indexes[0] += 1
	indexes[1] = (e.P-indexes[1])%e.P + 1
	indexes[2] += 1

	return
}

// GetBackwardNeighborIndexes returns the index of backward neighbors that can be entangled with current node
func (e *Entangler) GetBackwardNeighborIndexes(index int) (indexes []int) {
	if e.Alpha > 3 {
		util.ThrowError("alpha should equal 3")
	}

	// d_i is tangled with p_{h,i}
	pos := e.GetPositionCategory(index)
	var h, rh, lh int
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

	indexes = []int{h, rh, lh}

	return
}

// GetForwardNeighborIndexes returns the index of forward neighbors that is the entangled output of current node
func (e *Entangler) GetForwardNeighborIndexes(index int) (indexes []int) {
	if e.Alpha > 3 {
		util.ThrowError("alpha should equal 3")
	}

	// d_i creates entangled block p_{i,j}
	pos := e.GetPositionCategory(index)
	var h, rh, lh int
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

	indexes = []int{h, rh, lh}

	return
}

// CheckValid checks if the index is inside the lattice
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
