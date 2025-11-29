// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"errors"
	"sync"
)

// Blockchain - The canonical chain with full state execution.
type Blockchain struct {
	mu sync.RWMutex

	blocksByHash   map[Hash]*Block
	blocksByHeight map[BlockHeight]*Block
	head           *Block

	state    *StateDB
	executor *Executor
}

// NewBlockchain initializes the full chain engine.
func NewBlockchain(state *StateDB, exec *Executor) *Blockchain {
	return &Blockchain{
		blocksByHash:   make(map[Hash]*Block),
		blocksByHeight: make(map[BlockHeight]*Block),
		state:          state,
		executor:       exec,
	}
}

// Head returns the current tip block.
func (bc *Blockchain) Head() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.head
}

// AddBlock → FULL STATE VALIDATION + SNAPSHOT + REVERT
func (bc *Blockchain) AddBlock(b *Block) error {
	if b == nil {
		return errors.New("nil block")
	}
	if b.Header == nil {
		return errors.New("nil block header")
	}
	if err := b.ValidateBasic(); err != nil {
		return err
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Snapshot first → if block fails = revert to clean state
	snap := bc.state.Snapshot()

	// Genesis
	if b.Header.Height == 0 {
		if bc.head != nil {
			bc.state.RevertToSnapshot(snap)
			return errors.New("genesis already exists")
		}
		return bc.commitBlock(b)
	}

	// Check parent
	parent, ok := bc.blocksByHash[b.Header.ParentHash]
	if !ok || parent == nil {
		bc.state.RevertToSnapshot(snap)
		return errors.New("unknown parent block")
	}
	if b.Header.Height != parent.Header.Height+1 {
		bc.state.RevertToSnapshot(snap)
		return errors.New("invalid height")
	}

	// Execute all tx inside block
	_, err := bc.executor.ExecuteBlock(b)
	if err != nil {
		bc.state.RevertToSnapshot(snap)
		return err
	}

	// Verify agreed state root
	finalRoot := bc.state.StateRoot()
	if finalRoot != b.Header.StateRoot {
		bc.state.RevertToSnapshot(snap)
		return errors.New("state root mismatch")
	}

	// Accept & Commit to chain
	return bc.commitBlock(b)
}

// commitBlock → write block permanently to DB
func (bc *Blockchain) commitBlock(b *Block) error {
	h := b.Hash()
	bc.blocksByHash[h] = b
	bc.blocksByHeight[b.Header.Height] = b

	if bc.head == nil || b.Header.Height > bc.head.Header.Height {
		bc.head = b
	}

	return nil
}

// Getters ↓
func (bc *Blockchain) GetBlockByHash(h Hash) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.blocksByHash[h]
}

func (bc *Blockchain) GetBlockByHeight(h BlockHeight) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.blocksByHeight[h]
}