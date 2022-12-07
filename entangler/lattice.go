package entangler

import (
	"context"
	"ipfs-alpha-entanglement-code/util"
	"sync"

	"golang.org/x/xerrors"
)

type BlockGetter interface {
	GetData(index int) ([]byte, error)
	GetParity(index int, strand int) ([]byte, error)
}

type Lattice struct {
	*sync.Mutex

	Entangler
	DataBlocks   []*Block
	ParityBlocks [][]*Block

	Getter BlockGetter
	Once   sync.Once

	requestCounter uint

	SwitchDepth uint
}

// NewLattice creates a new lattice for block downloading and recovering
func NewLattice(alpha int, s int, p int, blockNum int, blockGetter BlockGetter, switchDepth uint) (lattice *Lattice) {
	var tangler = *NewEntangler(alpha, s, p)
	tangler.ChunkNum = blockNum
	lattice = &Lattice{
		Mutex:        &sync.Mutex{},
		Entangler:    tangler,
		DataBlocks:   make([]*Block, 0),
		ParityBlocks: make([][]*Block, alpha),
		Getter:       blockGetter,
		SwitchDepth:  switchDepth,
	}

	return
}

// Init inits the lattice by creating the entire structure in memory
func (l *Lattice) Init() {
	l.Once.Do(func() {
		// Create datablocks
		for i := 0; i < l.ChunkNum; i++ {
			datab := NewBlock(i+1, false)
			datab.LeftNeighbors = make([]*Block, l.Alpha)
			datab.RightNeighbors = make([]*Block, l.Alpha)
			l.DataBlocks = append(l.DataBlocks, datab)
		}

		// Create parities
		for k := 0; k < l.Alpha; k++ {
			for i := 0; i < l.ChunkNum; i++ {
				parityb := NewBlock(i+1, true)
				parityb.LeftNeighbors = make([]*Block, 1)
				parityb.RightNeighbors = make([]*Block, 1)
				parityb.Strand = k
				l.ParityBlocks[k] = append(l.ParityBlocks[k], parityb)
			}
		}

		// Link
		for i := 0; i < l.ChunkNum; i++ {
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
					if rightDataBlock != datab {
						rightDataBlock.RightNeighbors[k].IsWrapModified = true
					}
				}
				rightParity.RightNeighbors[0] = rightDataBlock
				rightDataBlock.LeftNeighbors[k] = rightParity
			}
		}
		util.LogPrint("Finish initializing lattice")
	})
}

// GetAllData returns all data in the data blocks as a byte array
func (l *Lattice) GetAllData() (data [][]byte, err error) {
	for i := 0; i < l.ChunkNum; i++ {
		var chunk []byte
		chunk, _, err = l.GetChunk(i + 1)
		if err != nil {
			return data, err
		}
		data = append(data, chunk)
	}

	return data, nil
}

// GetChunk returns a data chunk in the indexed block
func (l *Lattice) GetChunk(index int) (data []byte, repaired bool, err error) {
	block := l.getBlock(index)
	data, err = l.getDataFromBlock(block, l.SwitchDepth)
	repaired = block.IsRepaired()

	return data, repaired, err
}

// getBlock returns an original data block with the given index
func (l *Lattice) getBlock(index int) (block *Block) {
	block = l.DataBlocks[index-1]
	return block
}

// getDataFromBlock recovers a block with missing chunk using the lattice (hybrid, auto switch)
func (l *Lattice) getDataFromBlock(block *Block, allowDepth uint) ([]byte, error) {
	rid := l.getRequestID()
	if allowDepth > 0 {
		data, err := l.getDataFromBlockSequential(rid, block, allowDepth)
		if err == nil {
			return data, nil
		}
	}

	return l.getDataFromBlockParallel(rid, block)
}

// getDataFromBlockSequential recovers a block with missing chunk using the lattice (single thread)
func (l *Lattice) getDataFromBlockSequential(rid uint, block *Block, allowDepth uint) (data []byte, err error) {
	recursiveRecover := func(block *Block, allowDepth uint) {
		var repairSuccess bool = false
		var modifyState bool = true
		defer func() {
			if modifyState {
				block.FinishRepair(repairSuccess)
			}
		}()

		// if already has data or already visited
		if !block.StartRepair(context.Background(), rid) {
			modifyState = false
			return
		}

		// download data
		downloadErr := l.downloadBlock(block)
		if downloadErr == nil {
			repairSuccess = true
			if block.IsParity {
				util.LogPrint(util.Magenta("{Parallel} Index: %d, Parity: %t, Strand: %d downloaded successfully"), block.Index, block.IsParity, block.Strand)
				return
			}
			util.LogPrint("{Sequential} Index: %d, Parity: %t, Strand: %d downloaded successfully", block.Index, block.IsParity, block.Strand)
			return
		}
		util.LogPrint(util.Red("{Sequential} Index: %d, Parity: %t, Strand: %d downloaded fail"), block.Index, block.IsParity, block.Strand)

		// repair data
		if allowDepth == 0 {
			util.LogPrint(util.Red("{Sequential} Index: %d, Parity: %t, Strand: %d repaired fail"), block.Index, block.IsParity, block.Strand)
			return
		}
		pairs := block.GetRecoverPairs()
		if len(pairs) == 0 {
			util.LogPrint(util.Red("{Sequential} Index: %d, Parity: %t, Strand: %d repaired fail"), block.Index, block.IsParity, block.Strand)
			return
		}
		for _, mypair := range pairs {
			leftChunk, RepairErr := l.getDataFromBlockSequential(rid, mypair.Left, allowDepth-1)
			if RepairErr != nil {
				continue
			}

			rightChunk, RepairErr := l.getDataFromBlockSequential(rid, mypair.Right, allowDepth-1)
			if RepairErr != nil {
				continue
			}

			if block.Recover(leftChunk, rightChunk) == nil {
				util.LogPrint(util.Green("{Sequential} Index: %d, Parity: %t, Strand: %d repaired successfully"), block.Index, block.IsParity, block.Strand)
				repairSuccess = true
				return
			}
		}
		util.LogPrint(util.Red("{Sequential} Index: %d, Parity: %t, Strand: %d repaired fail"), block.Index, block.IsParity, block.Strand)
	}
	recursiveRecover(block, allowDepth)

	data, err = block.GetData()
	if err != nil {
		err = xerrors.Errorf("fail to recover block %d (parity: %t. strand: %d): %s.", block.Index, block.IsParity, block.Strand, err)
	}

	return data, err
}

// getDataFromBlockParallel recovers a block with missing chunk using the lattice (multiple threads)
func (l *Lattice) getDataFromBlockParallel(rid uint, block *Block) (data []byte, err error) {
	myCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var recursiveRecover func(context.Context, uint, *Block, chan bool)
	recursiveRecover = func(ctx context.Context, rid uint, block *Block, channel chan bool) {
		var repairSuccess bool = false
		var modifyState bool = true
		defer func() {
			if modifyState {
				block.FinishRepair(repairSuccess)
			}
			channel <- true
		}()

		select {
		case <-ctx.Done():
			return
		default:
			// if already has data or already visited
			if !block.StartRepair(ctx, rid) {
				modifyState = false
				return
			}

			// download data
			err := l.downloadBlock(block)
			if err == nil {
				repairSuccess = true
				if block.IsParity {
					util.LogPrint(util.Magenta("{Parallel} Index: %d, Parity: %t, Strand: %d downloaded successfully"), block.Index, block.IsParity, block.Strand)
					return
				}
				util.LogPrint("{Parallel} Index: %d, Parity: %t, Strand: %d downloaded successfully", block.Index, block.IsParity, block.Strand)
				return
			}
			util.LogPrint(util.Red("{Parallel} Index: %d, Parity: %t, Strand: %d downloaded fail"), block.Index, block.IsParity, block.Strand)

			// repair data
			pairs := block.GetRecoverPairs()
			if len(pairs) == 0 {
				util.LogPrint(util.Red("{Parallel} Index: %d, Parity: %t, Strand: %d repaired fail"), block.Index, block.IsParity, block.Strand)
				return
			}
			finish := make(chan bool)
			counter := 0
			for _, mypair := range pairs {
				util.InfoPrint(util.Yellow("{Parallel} Left - Index: %d, Parity: %t, Strand: %d\nRight - Index: %d, Parity: %t, Strand: %d\n\n"),
					mypair.Left.Index, mypair.Left.IsParity, mypair.Left.Strand,
					mypair.Right.Index, mypair.Right.IsParity, mypair.Right.Strand)
				go func(pair *BlockPair) {
					// tell the caller current func is finished
					success := false
					defer func() { finish <- success }()

					resultChan := make(chan bool, 2)
					go recursiveRecover(ctx, rid, pair.Left, resultChan)
					go recursiveRecover(ctx, rid, pair.Right, resultChan)

					<-resultChan
					<-resultChan
					leftChunk, err := pair.Left.GetData()
					if err != nil {
						return
					}
					// special case: wrap on itself
					if pair.Left == pair.Right {
						block.SetData(leftChunk, true)
						success = true
						return
					}
					rightChunk, err := pair.Right.GetData()
					if err != nil {
						return
					}

					if block.Recover(leftChunk, rightChunk) == nil {
						success = true
					}
				}(mypair)
			}
			// wait until one recover success, or all routine finishes
			for {
				success := <-finish
				if success {
					repairSuccess = true
					util.LogPrint(util.Green("{Parallel} Index: %d, Parity: %t, Strand: %d repaired successfully"), block.Index, block.IsParity, block.Strand)
					return
				}
				counter++
				if counter >= len(pairs) {
					util.LogPrint(util.Red("{Parallel} Index: %d, Parity: %t, Strand: %d repaired fail"), block.Index, block.IsParity, block.Strand)
					return
				}
			}
		}
	}

	myChannel := make(chan bool, 1)
	recursiveRecover(myCtx, rid, block, myChannel)
	<-myChannel
	data, err = block.GetData()
	if err != nil {
		err = xerrors.Errorf("fail to recover block %d (parity: %t. strand: %d): %s.", block.Index, block.IsParity, block.Strand, err)
	}

	return data, err
}

// downloadBlock downloads data/parity blocks using the Getter passed in
func (l *Lattice) downloadBlock(block *Block) (err error) {
	var data []byte
	if block.IsParity {
		data, err = l.Getter.GetParity(block.Index, block.Strand)
	} else {
		data, err = l.Getter.GetData(block.Index)
	}
	if err == nil {
		block.SetData(data, false)
	}

	return err
}

// generate uniq id for the request
func (l *Lattice) getRequestID() uint {
	l.Lock()
	defer l.Unlock()

	id := l.requestCounter
	l.requestCounter++
	return id
}
