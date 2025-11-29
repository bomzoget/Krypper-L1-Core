// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/big"
	"sort"
	"sync"
)

type StateDB struct {
	mu            sync.RWMutex
	accounts      map[Address]*Account
	snapshots     map[uint64]map[Address]*Account
	nextSnapshot  uint64
}

// NewStateDB creates a new empty state database.
func NewStateDB() *StateDB {
	return &StateDB{
		accounts:  make(map[Address]*Account),
		snapshots: make(map[uint64]map[Address]*Account),
	}
}

// internal: caller must hold lock.
func (s *StateDB) getOrCreate(addr Address) *Account {
	acc, ok := s.accounts[addr]
	if !ok {
		acc = NewAccount(addr)
		s.accounts[addr] = acc
	}
	return acc
}

// GetAccount returns a copy for read-only usage.
func (s *StateDB) GetAccount(addr Address) *Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, ok := s.accounts[addr]
	if !ok {
		return NewAccount(addr)
	}
	return acc.Copy()
}

// AddBalance adds amount to an address balance.
func (s *StateDB) AddBalance(addr Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.New("amount must be non-negative")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acc := s.getOrCreate(addr)
	return acc.AddBalance(amount)
}

// SubBalance subtracts amount from an address balance.
func (s *StateDB) SubBalance(addr Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.New("amount must be non-negative")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acc := s.getOrCreate(addr)
	return acc.SubBalance(amount)
}

// IncrementNonce increments account nonce.
func (s *StateDB) IncrementNonce(addr Address) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc := s.getOrCreate(addr)
	return acc.IncrementNonce()
}

// SetAccount overwrites the state for an address.
func (s *StateDB) SetAccount(acc *Account) error {
	if acc == nil {
		return errors.New("nil account")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	copyAcc := acc.Copy()
	s.accounts[copyAcc.Address] = copyAcc
	return nil
}

// Snapshot creates a deep copy snapshot and returns its id.
func (s *StateDB) Snapshot() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id := s.nextSnapshot
	s.nextSnapshot++

	snap := make(map[Address]*Account, len(s.accounts))
	for addr, acc := range s.accounts {
		snap[addr] = acc.Copy()
	}

	s.snapshots[id] = snap
	return id
}

// RevertToSnapshot restores the state to a previous snapshot.
func (s *StateDB) RevertToSnapshot(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, ok := s.snapshots[id]
	if !ok {
		return
	}

	restore := make(map[Address]*Account, len(snap))
	for addr, acc := range snap {
		restore[addr] = acc.Copy()
	}

	s.accounts = restore

	// optional: cleanup snapshot
	delete(s.snapshots, id)
}

// StateRoot computes a deterministic hash of all accounts.
func (s *StateDB) StateRoot() Hash {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.accounts) == 0 {
		return ZeroHash()
	}

	addrs := make([]Address, 0, len(s.accounts))
	for addr := range s.accounts {
		addrs = append(addrs, addr)
	}

	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i][:], addrs[j][:]) < 0
	})

	leaves := make([]Hash, 0, len(addrs))
	for _, addr := range addrs {
		acc := s.accounts[addr]
		h := sha256.Sum256(append(addr[:], acc.Hash()[:]...))
		leaves = append(leaves, Hash(h))
	}

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