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

// StateDB is an in-memory world state for KRYPPER L1.
// It supports snapshot/revert and deterministic root hashing.
type StateDB struct {
	mu        sync.RWMutex
	accounts  map[Address]*Account
	snapshotID uint64
	snapshots map[uint64]map[Address]*Account
}

// Constructor
func NewStateDB() *StateDB {
	return &StateDB{
		accounts:  make(map[Address]*Account),
		snapshots: make(map[uint64]map[Address]*Account),
	}
}

// =============== ACCOUNT ACCESS LAYER ===============

func (s *StateDB) getInternal(addr Address) *Account {
	acc, ok := s.accounts[addr]
	if !ok {
		acc = NewAccount(addr)
		s.accounts[addr] = acc
	}
	return acc
}

func (s *StateDB) GetAccount(addr Address) *Account {
	s.mu.RLock(); defer s.mu.RUnlock()
	acc, ok := s.accounts[addr]
	if !ok { return NewAccount(addr) }
	return acc.Copy()
}

func (s *StateDB) SetAccount(acc *Account) error {
	if acc == nil { return errors.New("nil account") }
	s.mu.Lock(); defer s.mu.Unlock()
	s.accounts[acc.Address] = acc.Copy()
	return nil
}

// =============== BALANCE / NONCE MUTATION ===============

func (s *StateDB) AddBalance(addr Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 { return errors.New("amount must be non-negative") }
	s.mu.Lock(); defer s.mu.Unlock()
	return s.getInternal(addr).AddBalance(amount)
}

func (s *StateDB) SubBalance(addr Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 { return errors.New("amount must be non-negative") }
	s.mu.Lock(); defer s.mu.Unlock()
	return s.getInternal(addr).SubBalance(amount)
}

func (s *StateDB) IncrementNonce(addr Address) error {
	s.mu.Lock(); defer s.mu.Unlock()
	return s.getInternal(addr).IncrementNonce()
}

// =============== SNAPSHOT / ROLLBACK ENGINE ===============

func (s *StateDB) Snapshot() uint64 {
	s.mu.RLock(); defer s.mu.RUnlock()

	s.snapshotID++
	id := s.snapshotID

	cp := make(map[Address]*Account, len(s.accounts))
	for a, acc := range s.accounts {
		cp[a] = acc.Copy()
	}
	s.snapshots[id] = cp
	return id
}

func (s *StateDB) RevertToSnapshot(id uint64) {
	s.mu.Lock(); defer s.mu.Unlock()

	snap, ok := s.snapshots[id]
	if !ok { return }

	restore := make(map[Address]*Account, len(snap))
	for a, acc := range snap {
		restore[a] = acc.Copy()
	}
	s.accounts = restore
	delete(s.snapshots, id)
}

func (s *StateDB) CommitSnapshot(id uint64) {
	s.mu.Lock(); defer s.mu.Unlock()
	delete(s.snapshots, id)
}

// =============== STATE ROOT / MERKLE TREE ===============

func (s *StateDB) StateRoot() Hash {
	s.mu.RLock(); defer s.mu.RUnlock()

	if len(s.accounts) == 0 { return ZeroHash() }

	addrs := make([]Address, 0, len(s.accounts))
	for a := range s.accounts { addrs = append(addrs, a) }

	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i][:], addrs[j][:]) < 0
	})

	leaves := make([]Hash, len(addrs))
	for i, a := range addrs {
		h := sha256.Sum256(append(a[:], s.accounts[a].Hash()[:]...))
		leaves[i] = Hash(h)
	}

	return merkleFromHashes(leaves)
}

func merkleFromHashes(h []Hash) Hash {
	if len(h) == 0 { return ZeroHash() }
	if len(h) == 1 { return h[0] }

	for len(h) > 1 {
		if len(h)%2 != 0 { h = append(h, h[len(h)-1]) }
		next := make([]Hash, 0, len(h)/2)
		for i := 0; i < len(h); i += 2 {
			s := sha256.Sum256(append(h[i][:], h[i+1][:]...))
			next = append(next, Hash(s))
		}
		h = next
	}
	return h[0]
}

// =============== PUBLIC READ HELPERS ===============

func (s *StateDB) GetBalance(addr Address) *big.Int {
	s.mu.RLock(); defer s.mu.RUnlock()
	acc, ok := s.accounts[addr]
	if !ok { return big.NewInt(0) }
	return new(big.Int).Set(acc.Balance)
}

func (s *StateDB) GetNonce(addr Address) uint64 {
	s.mu.RLock(); defer s.mu.RUnlock()
	acc, ok := s.accounts[addr]
	if !ok { return 0 }
	return acc.Nonce
}

// Mint is explicit inflation (used by rewards, consensus).
func (s *StateDB) Mint(addr Address, amount *big.Int) error {
	return s.AddBalance(addr, amount)
}