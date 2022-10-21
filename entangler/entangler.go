package entangler

import "fmt"

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

	cachedParityH  [][]byte
	cachedParityRH [][]byte
	cachedParityLH [][]byte
}

func NewEntangler(alpha int, s int, p int, chunkSize int) (entangler *Entangler) {
	entangler = &Entangler{Alpha: alpha, S: s, P: p, ChunkSize: chunkSize}
	entangler.cachedParityH = make([][]byte, s)
	entangler.cachedParityRH = make([][]byte, p)
	entangler.cachedParityLH = make([][]byte, p)

	return
}

// Entangle generate the entangelement for the given arrray of blocks
func (e *Entangler) Entangle(blocks [][]byte) (HParities, RHParities, LHParities []*EntangledBlock) {
	HParities = make([]*EntangledBlock, len(blocks))
	RHParities = make([]*EntangledBlock, len(blocks))
	LHParities = make([]*EntangledBlock, len(blocks))

	// generate the lattice
	for i, block := range blocks {
		hBlock, rBlock, lBlock := e.EntangleSingleBlock(i+1, block)
		HParities[i] = hBlock
		RHParities[i] = rBlock
		LHParities[i] = lBlock
	}

	// TODO: wraps the lattice

	return
}

// EntangleSingleBlock reads the backward parity neighbors from cache and produce the corresponding forward parity neighbors
func (e *Entangler) EntangleSingleBlock(index int, data []byte) (hBlock, rBlock, lBlock *EntangledBlock) {
	// read parity block from cache
	hCached, rCached, lCached := e.GetCachedPosition(index)
	hParityBytes := e.cachedParityH[hCached]
	rParityBytes := e.cachedParityRH[rCached]
	lParityBytes := e.cachedParityLH[lCached]

	// generate new parity block and cache
	hNext := e.XORBlockData(data, hParityBytes)
	e.cachedParityRH[hCached] = hNext
	rNext := e.XORBlockData(data, rParityBytes)
	e.cachedParityRH[rCached] = rNext
	lNext := e.XORBlockData(data, lParityBytes)
	e.cachedParityRH[lCached] = lNext

	// generate entangled block
	hIndex, rIndex, lIndex := e.GetForwardNeighborsIndex(index)
	hBlock = &EntangledBlock{
		LeftBlockIndex: index, RightBlockIndex: hIndex,
		Data: hNext, Strand: Horizontal}
	rBlock = &EntangledBlock{LeftBlockIndex: index, RightBlockIndex: rIndex,
		Data: rNext, Strand: Right}
	lBlock = &EntangledBlock{LeftBlockIndex: index, RightBlockIndex: lIndex,
		Data: lNext, Strand: Left}

	return
}

// GetPositionCategory determines which category the node belongs. Top, Bottom or Central
func (e *Entangler) GetPositionCategory(index int) PositionClass {
	nodePos := index % e.S
	if nodePos == -1 || nodePos == -4 {
		return Top
	} else if nodePos == 0 {
		return Bottom
	}
	return Central
}

// GetCachedPosition reads the cached backward parity neighbors of the current indexed node
func (e *Entangler) GetCachedPosition(index int) (h, rh, lh int) {
	h = (index - 1) % e.S

	indexInWindow := (index - 1) % (e.S * e.P)
	x := indexInWindow % e.P
	y := indexInWindow / e.P

	rh = (x - y + e.P) % e.P
	lh = (x + y) % e.S

	return
}

// GetBackwardNeighborsIndex returns the index of backward neighbors that can be entangled with current node
func (e *Entangler) GetBackwardNeighborsIndex(index int) (h, rh, lh int) {
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

// GetForwardNeighborsIndex returns the index of forward neighbors that is the entangled output of current node
func (e *Entangler) GetForwardNeighborsIndex(index int) (h, rh, lh int) {
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
