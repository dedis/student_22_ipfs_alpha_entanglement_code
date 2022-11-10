package entangler

import (
	"sync"

	"golang.org/x/xerrors"
)

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

type BlockStatus int

const (
	NoAttempt     BlockStatus = iota // Did not attempt to download or repair
	DataAvailable                    // Data already available
	RepairPending                    // Repair starts but not finish
	RepairFailed                     // Repair failed
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

	Status   BlockStatus
	IsParity bool
	Index    int

	// parity block parameters
	Strand         int
	IsWrapModified bool

	recoverPairs []*BlockPair
}

// NewBlock creates a block in the lattice
func NewBlock(index int, parityBlock bool) (block *Block) {
	block = &Block{
		RWMutex:  &sync.RWMutex{},
		Index:    index,
		IsParity: parityBlock,
		Status:   NoAttempt,
	}

	return
}

// IsAvailable checks whether the block has downloaded data
func (b *Block) IsAvailable() bool {
	b.RLock()
	defer b.RUnlock()

	return b.Status == DataAvailable
}

// IsRepairFailed checks whether the block is failed to be repaired
func (b *Block) IsRepairFailed() bool {
	b.RLock()
	defer b.RUnlock()

	return b.Status == RepairFailed
}

// HasNoAttempt checks whether there is any downloading / repairing happens on the block
func (b *Block) HasNoAttempt() bool {
	b.RLock()
	defer b.RUnlock()

	return b.Status == NoAttempt
}

// GetData returns the chunk data if available
func (b *Block) GetData() (data []byte, err error) {
	b.RLock()
	defer b.RUnlock()

	if b.Status == DataAvailable {
		data = b.Data
	} else {
		err = xerrors.Errorf("no available data")
	}

	return
}

// StartRepair sets the block's status to RepairPending
func (b *Block) StartRepair() {
	b.Lock()
	defer b.Unlock()
	b.Status = RepairPending
}

// SetData sets the chunk data inside the block
func (b *Block) SetData(data []byte) {
	if len(data) == 0 {
		return
	}

	if b.IsAvailable() {
		return
	}

	b.Lock()
	defer b.Unlock()

	b.Data = data
	b.Status = DataAvailable
}

// Recover recovers the block by xoring two given chunk
func (b *Block) Recover(v []byte, w []byte) (err error) {
	if len(v) == 0 || len(w) == 0 {
		b.Status = RepairFailed
		err = xerrors.Errorf("invalid recover input!")
		return
	}
	data := XORChunkData(v, w)
	b.SetData(data)
	return
}

// GetRecoverPairs returns a list of pair that can be used to do recovery
func (b *Block) GetRecoverPairs() (pairs []*BlockPair) {
	if b.recoverPairs != nil {
		pairs = b.recoverPairs
		return
	}

	if b.IsParity {
		// backward neighbors
		r := b.LeftNeighbors[0]
		l := r.LeftNeighbors[b.Strand]
		if l.IsWrapModified {
			l = l.LeftNeighbors[0]
		}
		pairs = append(pairs, &BlockPair{Left: l, Right: r})

		// forward neighbors
		l = r.RightNeighbors[0]
		r = l.RightNeighbors[b.Strand]
		if !b.IsWrapModified {
			pairs = append(pairs, &BlockPair{Left: l, Right: r})
		}
	} else {
		for k, _ := range b.LeftNeighbors {
			l := b.LeftNeighbors[k]
			r := b.RightNeighbors[k]
			if l.IsWrapModified {
				l = l.LeftNeighbors[0]
			} else if r.IsWrapModified {
				l = r.RightNeighbors[0]
				r = l.RightNeighbors[k]
			}
			pairs = append(pairs, &BlockPair{Left: l, Right: r})
		}
	}
	b.recoverPairs = pairs

	return
}
