package entangler

import (
	"context"
	"sync"

	"golang.org/x/xerrors"
)

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
	DataBlockNum int

	Getter BlockGetter
	Once   sync.Once
}

// NewLattice creates a new lattice for block downloading and recovering
func NewLattice(alpha int, s int, p int, blockSize int, blockGetter BlockGetter) (lattice *Lattice) {
	var once sync.Once
	lattice = &Lattice{
		Entangler:    *NewEntangler(alpha, s, p),
		DataBlocks:   make([]*Block, 0),
		ParityBlocks: make([][]*Block, alpha),
		Getter:       blockGetter,
		Once:         once,
	}

	return
}

// Init inits the lattice by creating the entire structure in memory
func (l *Lattice) Init() {
	l.Once.Do(func() {
		// Create datablocks
		for i := 0; i < l.DataBlockNum; i++ {
			datab := NewBlock(i+1, false)
			datab.LeftNeighbors = make([]*Block, l.Alpha)
			datab.RightNeighbors = make([]*Block, l.Alpha)
			l.DataBlocks = append(l.DataBlocks, datab)
		}

		// Create parities
		for k := 0; k < l.Alpha; k++ {
			for i := 0; i < l.DataBlockNum; i++ {
				parityb := NewBlock(i+1, true)
				parityb.LeftNeighbors = make([]*Block, 1)
				parityb.RightNeighbors = make([]*Block, 1)
				parityb.Strand = k
				l.ParityBlocks[k] = append(l.ParityBlocks[k], parityb)
			}
		}

		// Link
		for i := 0; i < l.DataBlockNum; i++ {
			datab := l.DataBlocks[i]
			forward := l.getForwardNeighborIndexes(i + 1)
			for k := 0; k < l.Alpha; k++ {
				rightParity := l.ParityBlocks[k][i]
				rightParity.LeftNeighbors[0] = datab
				datab.RightNeighbors[k] = rightParity

				var rightDataBlock *Block
				if l.IsValidIndex(forward[k]) {
					rightDataBlock = l.DataBlocks[forward[k]-1]
				} else {
					// Wrap lattice
					index := l.getChainStartIndexes(i + 1)[k]
					rightDataBlock = l.DataBlocks[index-1]
					rightDataBlock.RightNeighbors[k].IsWrapModified = true
				}
				rightParity.RightNeighbors[0] = rightDataBlock
				rightDataBlock.LeftNeighbors[k] = rightParity
			}
		}

	})
}

// GetAllData returns all data in the data blocks as a byte array
func (l *Lattice) GetAllData() (data []byte, err error) {
	for i := 0; i < l.DataBlockNum; i++ {
		var chunk []byte
		chunk, err = l.GetChunk(i + 1)
		if err != nil {
			return
		}
		data = append(data, chunk...)
	}

	return
}

// GetChunk returns a data chunk in the indexed block
func (l *Lattice) GetChunk(index int) (data []byte, err error) {
	block := l.getBlock(index)
	data, err = l.getDataFromBlock(block)

	return
}

// getBlock returns an original data block with the given index
func (l *Lattice) getBlock(index int) (block *Block) {
	block = l.DataBlocks[index-1]
	return
}

// getDataFromBlock recovers a block with missing chunk using the lattice
func (l *Lattice) getDataFromBlock(block *Block) (data []byte, err error) {
	myCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var recursiveRecover func(*Block, context.Context, chan []byte)
	recursiveRecover = func(block *Block, ctx context.Context, channel chan []byte) {
		defer func() { channel <- block.GetData() }()

		select {
		case <-ctx.Done():
			return
		default:
			// if already tried to download or repair
			if !block.HasNoAttempt() {
				return
			} else {
				block.StartRepair()
			}

			// download data
			data, err := l.Getter.GetData(block.Index)
			if err == nil {
				block.SetData(data)
				return
			}

			// repair data
			success := make(chan bool)
			finish := make(chan bool)
			counter := 0
			pairs := block.GetRecoverPairs()
			for _, mypair := range pairs {
				go func(pair *BlockPair, ctx context.Context) {
					// tell the caller current func is finished
					defer func() { finish <- true }()
					resultChan := make(chan []byte, 2)
					go recursiveRecover(pair.Left, ctx, resultChan)
					go recursiveRecover(pair.Right, ctx, resultChan)

					if block.Recover(<-resultChan, <-resultChan) == nil {
						success <- true
					}
				}(mypair, ctx)
			}
			// wait until one recover success, or all routine finishes
			for {
				select {
				case <-success:
					return
				case <-finish:
					counter++
					if counter >= len(pairs) {
						return
					}
				}
			}
		}
	}

	myChannel := make(chan []byte, 1)
	recursiveRecover(block, myCtx, myChannel)
	data = <-myChannel
	if len(data) > 0 {
		err = xerrors.Errorf("fail to recover the block")
	}

	return
}
