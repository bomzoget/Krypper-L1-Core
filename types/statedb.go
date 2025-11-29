// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"bytes"        // Faster byte comparison
	"crypto/sha256"
	"errors"
	"math/big"
	"sort"
	"sync"
)

/*
	StateDB — In-memory deterministic world state
	Keeps Address → Account mapping + produces StateRoot for block headers.
	Thread-safe (RWMutex) & optimized for L1 prototype phase.
*/

type StateDB struct {
	mu       sync.RWMutex
	accounts map[Address]*Account
}

// Create blank state
func NewStateDB() *StateDB {
	return &StateDB{accounts: make(map[Address]*Account)}
}

// Internal — requires lock
func (s *StateDB) getInternal(addr Address) *Account {
	acc, ok := s.accounts[addr]         // ← 🔥 fixed syntax OK
	if !ok {
		acc = NewAccount(addr)
		s.accounts[addr] = acc
	}
	return acc
}

// Read-only copy
func (s *StateDB) GetAccount(addr Address) *Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if acc, ok := s.accounts[addr]; ok {
		return acc.Copy()
	}
	return NewAccount(addr)
}

// ---- BALANCE OPS ----

func (s *StateDB) AddBalance(addr Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.New("amount must be non-negative")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getInternal(addr).AddBalance(amount)
}

func (s *StateDB) SubBalance(addr Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.New("amount must be non-negative")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getInternal(addr).SubBalance(amount)
}

func (s *StateDB) IncrementNonce(addr Address) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getInternal(addr).IncrementNonce()
}

// Replace entire account safely
func (s *StateDB) SetAccount(acc *Account) error {
	if acc == nil {
		return errors.New("nil account")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[acc.Address] = acc.Copy()
	return nil
}

// ---- 🔥 StateRoot (Deterministic Merkle) ----

func (s *StateDB) StateRoot() Hash {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.accounts) == 0 {
		return ZeroHash()
	}

	// 1) sorted keys for determinism
	addrs := make([]Address, 0, len(s.accounts))
	for a := range s.accounts {
		addrs = append(addrs, a)
	}

	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i][:], addrs[j][:]) < 0
	})

	// 2) build leaf hashes = H(addr||accountHash)
	leaves := make([]Hash, 0, len(addrs))
	for _, a := range addrs {
		acc := s.accounts[a]
		h := sha256.Sum256(append(a[:], acc.Hash()[:]...))
		leaves = append(leaves, Hash(h))
	}

	// 3) compute merkle root
	return merkleFromHashes(leaves)
}

func merkleFromHashes(leaves []Hash) Hash {
	if len(leaves) == 0 {
		return ZeroHash()
	}
	if len(leaves) == 1 {
		return leaves[0]
	}

	hashes := make([]Hash, len(leaves))
	copy(hashes, leaves)

	for len(hashes) > 1 {
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		next := make([]Hash, 0, len(hashes)/2)
		for i := 0; i < len(hashes); i += 2 {
			sum := sha256.Sum256(append(hashes[i][:], hashes[i+1][:]...))
			next = append(next, Hash(sum))
		}
		hashes = next
	}
	return hashes[0]
}