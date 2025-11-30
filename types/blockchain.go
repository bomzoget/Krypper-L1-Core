// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"errors"
	"sync"
)

// Blockchain manages blocks, verifies transitions, commits state.
type Blockchain struct {
	mu             sync.RWMutex
	state          *StateDB
	executor       *Executor
	blocksByHash   map[Hash]*Block
	blocksByHeight map[uint64]*Block
	head           *Block
}

// NewBlockchain creates a chain with the given StateDB and Executor.
func NewBlockchain(state *StateDB, executor *Executor) *Blockchain {
	return &Blockchain{
		state:          state,
		executor:       executor,
		blocksByHash:   make(map[Hash]*Block),
		blocksByHeight: make(map[uint64]*Block),
		head:           nil,
	}
}

// Head returns the current tip of the chain.
func (bc *Blockchain) Head() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.head
}

// GetBlockByHash returns a block by its hash, or nil if not found.
func (bc *Blockchain) GetBlockByHash(h Hash) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.blocksByHash[h]
}

// GetBlockByHeight returns a block by its height, or nil if not found.
func (bc *Blockchain) GetBlockByHeight(height uint64) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.blocksByHeight[height]
}

// AddBlock validates, executes, and commits a new block atomically.
func (bc *Blockchain) AddBlock(b *Block) error {
	if b == nil {
		return errors.New("nil block")
	}

	// Stateless checks (header + tx root + tx basic)
	if err := b.ValidateBasic(); err != nil {
		return err
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Take a global snapshot for this block.
	blockSnap := bc.state.Snapshot()

	// ------------------------------------------------------------
	// GENESIS BLOCK (HEIGHT 0)
	// ------------------------------------------------------------
	if b.Header.Height == 0 {
		if bc.head != nil {
			bc.state.RevertToSnapshot(blockSnap)
			return errors.New("genesis already exists")
		}

		// Set coinbase to proposer for any potential tx fees.
		bc.executor.SetCoinbase(b.Header.Proposer)

		// Execute genesis transactions if any.
		if len(b.Transactions) > 0 {
			if _, err := bc.executor.ExecuteBlock(b); err != nil {
				bc.state.RevertToSnapshot(blockSnap)
				return err
			}
		}

		// Verify state root after execution.
		finalRoot := bc.state.StateRoot()
		if finalRoot != b.Header.StateRoot {
			bc.state.RevertToSnapshot(blockSnap)
			return errors.New("genesis state mismatch")
		}

		// Success: commit snapshot and index block.
		bc.state.CommitSnapshot(blockSnap)
		return bc.commitBlock(b)
	}

	// ------------------------------------------------------------
	// NORMAL BLOCK
	// ------------------------------------------------------------

	// Check parent existence.
	parent, ok := bc.blocksByHash[b.Header.ParentHash]
	if !ok || parent == nil {
		bc.state.RevertToSnapshot(blockSnap)
		return errors.New("unknown parent block")
	}

	// Check height continuity.
	if b.Header.Height != parent.Header.Height+1 {
		bc.state.RevertToSnapshot(blockSnap)
		return errors.New("invalid height")
	}

	// Set coinbase for fee distribution.
	bc.executor.SetCoinbase(b.Header.Proposer)

	// Execute all transactions.
	if _, err := bc.executor.ExecuteBlock(b); err != nil {
		bc.state.RevertToSnapshot(blockSnap)
		return err
	}

	// Verify state root matches header.
	finalRoot := bc.state.StateRoot()
	if finalRoot != b.Header.StateRoot {
		bc.state.RevertToSnapshot(blockSnap)
		return errors.New("state root mismatch")
	}

	// Success: commit snapshot and index block.
	bc.state.CommitSnapshot(blockSnap)
	return bc.commitBlock(b)
}

// commitBlock writes the block into indexes and moves head forward.
// Caller must hold bc.mu (write lock).
func (bc *Blockchain) commitBlock(b *Block) error {
	h := b.Hash()
	bc.blocksByHash[h] = b
	bc.blocksByHeight[uint64(b.Header.Height)] = b
	bc.head = b
	return nil
}