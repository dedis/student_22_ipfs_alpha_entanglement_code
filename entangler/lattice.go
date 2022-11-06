package entangler

type BlockGetter interface {
	GetData(index int) ([]byte, error)
}

type Lattice struct {
	// Alpha     int // TODO: now only support alpha = 3 ???
	// S         int
	// P         int
	// BlockSize int
	Entangler
	DataBlocks   []*Block
	ParityBlocks [][]*Block

	DataBlockNum         int
	MissingDataBlocksNum int

	Getter BlockGetter
}

// NewLattice creates a new lattice for block downloading and recovering
func NewLattice(alpha int, s int, p int, blockSize int, blockGetter BlockGetter) (lattice *Lattice) {
	lattice = &Lattice{
		Entangler:    *NewEntangler(alpha, s, p),
		DataBlocks:   make([]*Block, 0),
		ParityBlocks: make([][]*Block, alpha),
		Getter:       blockGetter,
	}

	return
}

// GetChunk returns a data chunk in the indexed block
func (l *Lattice) GetChunk(index int) (data []byte, err error) {
	block := l.getBlock(index)
	err = l.recoverBlock(block)
	if err == nil {
		data = block.GetData()
	}

	return
}

// getBlock returns an original data block with the given index
func (l *Lattice) getBlock(index int) (block *Block) {
	block = l.DataBlocks[index-1]
	return
}

// recoverBlock recovers a block with missing chunk using the lattice
func (l *Lattice) recoverBlock(block *Block) (err error) {
	// if already has data, return
	if block.IsAvailable() {
		return
	}

	// download data
	data, err := l.Getter.GetData(block.Index)
	if err == nil {
		block.SetData(data)
		return
	}

	// repair data
	pairs := block.GetRecoverPairs()
	for _, pair := range pairs {
		if block.IsAvailable() {
			return
		}

		err = l.recoverBlock(pair.Left)
		if err != nil {
			continue
		}
		err = l.recoverBlock(pair.Right)
		if err != nil {
			continue
		}

		err = block.Recover(pair.Left.Data, pair.Right.Data)
		if err == nil {
			return
		}
	}

	return
}
