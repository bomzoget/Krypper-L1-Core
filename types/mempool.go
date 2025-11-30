// SPDX-License-Identifier: MIT
// Dev: KryperAI Final Mempool

package types

import (
	"errors"
	"math/big"
	"sort"
	"sync"
)

type Mempool struct {
	mu      sync.RWMutex
	pending []*Transaction
	state   *StateDB
	maxSize int
}

func NewMempool(state *StateDB) *Mempool {
	return &Mempool{
		state:   state,
		maxSize: 5000,
		pending: make([]*Transaction, 0),
	}
}

func (m *Mempool) AddTx(tx *Transaction) error {
	if tx == nil {
		return errors.New("nil tx")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Detect duplicate
	txHash := tx.Hash()
	for _, t := range m.pending {
		if t.Hash() == txHash {
			return errors.New("duplicate transaction")
		}
	}

	// Recover signer = signature verification
	from, err := RecoverTxSender(tx)
	if err != nil {
		return errors.New("invalid signature")
	}

	// Balance check
	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit), tx.GasPrice)
	totalCost := new(big.Int).Add(tx.Value, gasCost)

	if m.state.GetBalance(from).Cmp(totalCost) < 0 {
		return errors.New("insufficient balance")
	}

	// Nonce check
	currentNonce := m.state.GetNonce(from)
	if tx.Nonce < currentNonce {
		return errors.New("nonce too low")
	}

	// Anti-spam full pool
	if len(m.pending) >= m.maxSize {
		m.evictLowestGas()
	}

	m.pending = append(m.pending, tx)
	return nil
}

func (m *Mempool) PopForBlock(n int) []*Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.pending) == 0 {
		return nil
	}

	sort.Slice(m.pending, func(i, j int) bool {
		return m.pending[i].GasPrice.Cmp(m.pending[j].GasPrice) > 0
	})

	if n > len(m.pending) {
		n = len(m.pending)
	}

	selected := m.pending[:n]
	m.pending = m.pending[n:]

	return selected
}

func (m *Mempool) evictLowestGas() {
	if len(m.pending) == 0 {
		return
	}
	sort.Slice(m.pending, func(i, j int) bool {
		return m.pending[i].GasPrice.Cmp(m.pending[j].GasPrice) < 0
	})
	m.pending = m.pending[1:]
}

func (m *Mempool) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pending)
}