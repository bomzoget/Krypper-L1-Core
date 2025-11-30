// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

import (
	"errors"
	"math/big"
	"sort"
	"sync"
)

type Mempool struct {
	mu        sync.RWMutex
	pending   []*Transaction
	state     *StateDB
	maxSize   int
}

// NewMempool initializes mempool
func NewMempool(state *StateDB) *Mempool {
	return &Mempool{
		state:   state,
		maxSize: 5000, // anti spam
		pending: make([]*Transaction, 0),
	}
}

// AddTx verifies + stores tx
func (m *Mempool) AddTx(tx *Transaction) error {
	if tx == nil {
		return errors.New("nil tx")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify signature before anything
	from, err := VerifyTxSignature(tx)
	if err != nil {
		return errors.New("invalid signature")
	}

	// Balance check for gas
	required := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit), tx.GasPrice)
	totalCost := new(big.Int).Add(required, tx.Value)

	if m.state.GetBalance(from).Cmp(totalCost) < 0 {
		return errors.New("insufficient balance for tx + gas")
	}

	// Nonce check
	currentNonce := m.state.GetNonce(from)
	if tx.Nonce < currentNonce {
		return errors.New("nonce too low (replay suspected)")
	}

	// Max size protection
	if len(m.pending) >= m.maxSize {
		m.evictLowestGas()
	}

	m.pending = append(m.pending, tx)
	return nil
}

// PopForBlock returns N best txs by GasPrice and removes them
func (m *Mempool) PopForBlock(n int) []*Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.pending) == 0 {
		return nil
	}

	// Highest gas first
	sort.Slice(m.pending, func(i, j int) bool {
		return m.pending[i].GasPrice.Cmp(m.pending[j].GasPrice) > 0
	})

	if n > len(m.pending) {
		n = len(m.pending)
	}

	selected := m.pending[:n]
	m.pending = m.pending[n:] // remove from pool

	return selected
}

// Drop tx with lowest gas when pool full
func (m *Mempool) evictLowestGas() {
	if len(m.pending) == 0 {
		return
	}
	sort.Slice(m.pending, func(i, j int) bool {
		return m.pending[i].GasPrice.Cmp(m.pending[j].GasPrice) < 0
	})
	m.pending = m.pending[1:]
}

// Count returns pending size
func (m *Mempool) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pending)
}