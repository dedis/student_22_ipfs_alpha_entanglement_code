package entangler

import (
	"context"
	"ipfs-alpha-entanglement-code/util"
	"sync"

	"golang.org/x/xerrors"
)

type BlockStatus int

const (
	NoData        BlockStatus = iota // Did not attempt to download or repair
	DataAvailable                    // Data already available
	RepairPending                    // Repair starts but not finish
)

// BlockPair records a pair of block
type BlockPair struct {
	Right, Left *Block
}

// Block is the data/parity block in lattice
type Block struct {
	*sync.RWMutex

	Data           []byte
	LeftNeighbors  []*Block
	RightNeighbors []*Block
	IsParity       bool
	Index          int
	Repaired       bool

	// parity block parameters
	Strand         int
	IsWrapModified bool

	recoverPairs []*BlockPair
	once         sync.Once

	Status       BlockStatus
	waitingGroup *sync.Cond
}

// NewBlock creates a block in the lattice
func NewBlock(index int, parityBlock bool) (block *Block) {
	m := sync.RWMutex{}
	var once sync.Once
	block = &Block{
		RWMutex:      &m,
		Index:        index,
		IsParity:     parityBlock,
		Status:       NoData,
		Repaired:     false,
		once:         once,
		waitingGroup: sync.NewCond(&m),
	}

	return
}

// GetData returns the chunk data if available
func (b *Block) GetData() (data []byte, err error) {
	b.RLock()
	defer b.RUnlock()

	if b.Status == DataAvailable {
		data = b.Data
	} else {
		err = xerrors.Errorf("no available data. Status: %d", b.Status)
	}

	return
}

func (b *Block) IsRepaired() bool {
	b.RLock()
	b.RUnlock()

	return b.Repaired
}

// StartRepair sets the block's status to RepairPending if no previous attempt
func (b *Block) StartRepair(ctx context.Context) bool {
	b.Lock()
	defer b.Unlock()
	for {
		select {
		case <-ctx.Done():
			return false
		default:
			if b.Status == RepairPending {
				b.waitingGroup.Wait()
			} else if b.Status == DataAvailable {
				return false
			} else {
				b.Status = RepairPending
				return true
			}
		}
	}
}

// FinishRepair update the block status and wake the waiting thread
func (b *Block) FinishRepair(success bool) {
	b.Lock()
	defer b.Unlock()
	shouldUpdate := b.Status == RepairPending
	if success {
		if shouldUpdate {
			b.Status = DataAvailable
		}
		b.waitingGroup.Broadcast()
	} else {
		if shouldUpdate {
			b.Status = NoData
		}
		b.waitingGroup.Signal()
	}
}

// SetData sets the chunk data inside the block
func (b *Block) SetData(data []byte) {
	b.Lock()
	defer b.Unlock()

	if b.Status != DataAvailable {
		b.Data = data
		b.Status = DataAvailable
	}
}

// Recover recovers the block by xoring two given chunk
func (b *Block) Recover(v []byte, w []byte) (err error) {
	if len(v) == 0 || len(w) == 0 {
		err = xerrors.Errorf("invalid recover input!")
		return
	}
	data := XORChunkData(v, w)

	b.Lock()
	defer b.Unlock()

	if !(b.Status == RepairPending || b.Status == DataAvailable) {
		util.LogPrint(util.Magenta("Status %d: Block: %d, Parity: %t, Strand %d"), b.Status, b.Index, b.IsParity, b.Strand)
	}

	if b.Status != DataAvailable {
		b.Data = data
		b.Status = DataAvailable
		b.Repaired = true
	}

	return
}

// GetRecoverPairs returns a list of pair that can be used to do recovery
func (b *Block) GetRecoverPairs() (pairs []*BlockPair) {
	b.once.Do(func() {
		if b.IsParity {
			// backward neighbors
			r := b.LeftNeighbors[0]
			l := r.LeftNeighbors[b.Strand]
			if l.IsWrapModified {
				l = l.LeftNeighbors[0]
			}
			pairs = append(pairs, &BlockPair{Left: l, Right: r})

			// forward neighbors
			l = b.RightNeighbors[0]
			r = l.RightNeighbors[b.Strand]
			if !b.IsWrapModified {
				pairs = append(pairs, &BlockPair{Left: l, Right: r})
			}
		} else {
			for k := range b.LeftNeighbors {
				l := b.LeftNeighbors[k]
				r := b.RightNeighbors[k]
				if l.IsWrapModified {
					l = l.LeftNeighbors[0]
				}
				pairs = append(pairs, &BlockPair{Left: l, Right: r})
				if r.IsWrapModified {
					l = r.RightNeighbors[0]
					r = l.RightNeighbors[k]
					pairs = append(pairs, &BlockPair{Left: l, Right: r})
				}
			}
		}
		b.recoverPairs = pairs
	})

	pairs = b.recoverPairs
	return
}

// XORChunkData pads the bytes to the desired length and XOR these two bytes array
func XORChunkData(chunk1 []byte, chunk2 []byte) (result []byte) {
	if len(chunk1) == 0 {
		return chunk2
	}
	if len(chunk2) == 0 {
		return chunk1
	}

	PaddingData(&chunk1, &chunk2)

	result = make([]byte, len(chunk1))
	for i := 0; i < len(chunk1); i++ {
		result[i] = chunk1[i] ^ chunk2[i]
	}

	return
}

// PaddingData pads the two chunks to the same length
func PaddingData(chunk1 *[]byte, chunk2 *[]byte) {
	len1, len2 := len(*chunk1), len(*chunk2)
	if len1 > len2 {
		padded := make([]byte, len1)
		copy(padded, *chunk2)
		*chunk2 = padded
	} else if len1 < len2 {
		padded := make([]byte, len2)
		copy(padded, *chunk1)
		*chunk1 = padded
	}
}
