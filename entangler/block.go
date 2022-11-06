package entangler

import (
	"sync"
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

// BlockPair records a pair of block
type BlockPair struct {
	Right, Left *Block
}

// Block is the data/parity block in lattice
type Block struct {
	*sync.RWMutex

	Data      []byte
	Neighbors []*BlockPair

	HasData  bool
	IsParity bool
	Index    int

	// parity block parameters
	Strand         int
	IsWrapModified bool

	recoverPairs []*BlockPair
}

// NewBlock creates a block in the lattice
func NewBlock(parityBlock bool) (block *Block) {
	block = &Block{
		IsParity: parityBlock,
		HasData:  false,
	}

	return
}

// IsAvailable checks whether the block has downloaded data
func (b *Block) IsAvailable() bool {
	b.RLock()
	defer b.RUnlock()

	return b.HasData
}

// GetData returns the chunk data if available
func (b *Block) GetData() (data []byte) {
	b.RLock()
	defer b.RUnlock()

	if b.HasData {
		data = b.Data
	}
	return
}

// SetData sets the chunk data inside the block
func (b *Block) SetData(data []byte) {
	if b.IsAvailable() {
		return
	}

	b.Lock()
	defer b.Unlock()

	b.Data = data
	b.HasData = true
}

// Recover recovers the block by xoring two given chunk
func (b *Block) Recover(v []byte, w []byte) (err error) {
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
		r := b.Neighbors[0].Left
		l := r.Neighbors[b.Strand].Left
		if l.IsWrapModified {
			l = l.Neighbors[0].Left
		}
		pairs = append(pairs, &BlockPair{Left: l, Right: r})

		// forward neighbors
		l = r.Neighbors[0].Right
		r = l.Neighbors[b.Strand].Right
		if !b.IsWrapModified {
			pairs = append(pairs, &BlockPair{Left: l, Right: r})
		}
	} else {
		for strand, pair := range b.recoverPairs {
			l := pair.Left
			r := pair.Right
			if l.IsWrapModified {
				l = l.Neighbors[0].Left
			} else if r.IsWrapModified {
				l = r.Neighbors[0].Right
				r = l.Neighbors[strand].Right
			}
			pairs = append(pairs, &BlockPair{Left: l, Right: r})
		}
	}
	b.recoverPairs = pairs

	return
}
