// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

import (
	"crypto/sha256"
	"encoding/binary"
)

// BlockHeader supports Tier1/Tier2/Tier3 consensus
type BlockHeader struct {
	ParentHash Hash
	Height     uint64
	Timestamp  int64
	StateRoot  Hash
	TxRoot     Hash
	GasLimit   uint64

	// Tier-based block production
	Proposer  Address // Tier1
	Validator Address // Tier2
	Witness   Address // Tier3
}

// HashHeader returns hash of block header
func (h *BlockHeader) HashHeader() Hash {
	b := sha256.New()
	var buf [8]byte

	b.Write(h.ParentHash[:])

	binary.BigEndian.PutUint64(buf[:], h.Height)
	b.Write(buf[:])

	binary.BigEndian.PutUint64(buf[:], uint64(h.Timestamp))
	b.Write(buf[:])

	b.Write(h.StateRoot[:])
	b.Write(h.TxRoot[:])

	binary.BigEndian.PutUint64(buf[:], h.GasLimit)
	b.Write(buf[:])

	b.Write(h.Proposer[:])
	b.Write(h.Validator[:])
	b.Write(h.Witness[:])

	var out Hash
	copy(out[:], b.Sum(nil))
	return out
}

// -------------------------------------------------------------

// Block is full block container
type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
	hash         Hash
}

// NewBlock constructs new block
func NewBlock(h *BlockHeader, txs []*Transaction) *Block {
	return &Block{Header: h, Transactions: txs}
}

// Hash returns block hash == header hash
func (b *Block) Hash() Hash {
	if !b.hash.IsZero() {
		return b.hash
	}
	b.hash = b.Header.HashHeader()
	return b.hash
}

// ComputeTxRoot calculates merkle-like root of txs
func (b *Block) ComputeTxRoot() {
	if len(b.Transactions) == 0 {
		b.Header.TxRoot = ZeroHash()
		return
	}
	h := make([]Hash, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		h = append(h, tx.Hash())
	}
	b.Header.TxRoot = merkleFromHashes(h)
}