// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"errors"
)

// Block is the full block object containing header and transaction list.
type Block struct {
	Header       BlockHeader    `json:"header"`
	Transactions []*Transaction `json:"transactions"`

	hash Hash `json:"-"`
}

// NewBlock creates a new block from header and tx list.
// Caller is expected to call ComputeTxRoot before finalizing the header hash.
func NewBlock(header BlockHeader, txs []*Transaction) *Block {
	return &Block{
		Header:       header,
		Transactions: txs,
	}
}

// Hash returns the block hash (hash of the header).
// Value is cached after first computation.
func (b *Block) Hash() Hash {
	if !b.hash.IsZero() {
		return b.hash
	}
	h := b.Header.HashHeader()
	b.hash = h
	return h
}

// ComputeTxRoot computes and sets the TxRoot field on the header
// using a Merkle tree over the transaction hashes.
func (b *Block) ComputeTxRoot() {
	if len(b.Transactions) == 0 {
		b.Header.TxRoot = ZeroHash()
		return
	}

	leaves := make([]Hash, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		if tx == nil {
			continue
		}
		leaves = append(leaves, tx.Hash())
	}

	if len(leaves) == 0 {
		b.Header.TxRoot = ZeroHash()
		return
	}

	root := merkleFromHashes(leaves)
	b.Header.TxRoot = root
}

// ValidateBasic performs stateless checks over the block and its transactions.
// It does not touch StateDB and does not verify signatures at state level.
func (b *Block) ValidateBasic() error {
	if b == nil {
		return errors.New("nil block")
	}

	// Header-level checks are assumed to be done in HashHeader/consensus layer,
	// but we can still check simple invariants here if needed.

	// Verify TxRoot matches the actual transaction list.
	calculatedRoot := ZeroHash()
	if len(b.Transactions) > 0 {
		leaves := make([]Hash, 0, len(b.Transactions))
		for _, tx := range b.Transactions {
			if tx == nil {
				return errors.New("nil transaction in block")
			}
			if err := tx.ValidateBasic(); err != nil {
				return err
			}
			leaves = append(leaves, tx.Hash())
		}
		calculatedRoot = merkleFromHashes(leaves)
	}

	if b.Header.TxRoot != calculatedRoot {
		return errors.New("tx root mismatch")
	}

	return nil
}