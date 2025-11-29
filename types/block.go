// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import "errors"

// ValidateBasic performs stateless checks on a block.
func (b *Block) ValidateBasic() error {
	if b == nil {
		return errors.New("nil block")
	}
	if b.Header == nil {
		return errors.New("nil block header")
	}

	h := b.Header

	if h.GasLimit == 0 {
		return errors.New("gasLimit must be > 0")
	}
	if h.GasUsed > h.GasLimit {
		return errors.New("gasUsed greater than gasLimit")
	}
	if h.Timestamp <= 0 {
		return errors.New("invalid timestamp")
	}

	// Recompute TxRoot from body and compare with header
	calculated := CalculateTxRoot(b.Transactions)
	if calculated != h.TxRoot {
		return errors.New("tx root mismatch")
	}

	return nil
}

// ComputeTxRoot recalculates the transaction root and updates the header.
// Use when building a new block before sealing.
func (b *Block) ComputeTxRoot() {
	if b == nil || b.Header == nil {
		return
	}
	b.Header.TxRoot = CalculateTxRoot(b.Transactions)
}