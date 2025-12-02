// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

import (
        "crypto/sha256"
        "math/big"
)

// StateDB is the chain global state container.
// In final implementation this should connect to a persistent DB (LevelDB/MPT),
// but for genesis & bring-up it works fully in-memory.
type StateDB struct {
        accounts  map[Address]*Account
        snapshots []map[Address]*Account
}

func NewStateDB() *StateDB {
        return &StateDB{
                accounts:  make(map[Address]*Account),
                snapshots: make([]map[Address]*Account, 0),
        }
}

// GetAccount returns an existing account or nil.
func (s *StateDB) GetAccount(addr Address) *Account {
        return s.accounts[addr]
}

// CreateAccount ensures a new account exists.
func (s *StateDB) CreateAccount(addr Address) error {
        if _, ok := s.accounts[addr]; ok {
                return nil
        }
        s.accounts[addr] = NewAccount(addr)
        return nil
}

// GetBalance returns the balance of an account.
func (s *StateDB) GetBalance(addr Address) *big.Int {
        acc := s.GetAccount(addr)
        if acc == nil {
                return big.NewInt(0)
        }
        return new(big.Int).Set(acc.Balance)
}

// GetNonce returns the nonce of an account.
func (s *StateDB) GetNonce(addr Address) uint64 {
        acc := s.GetAccount(addr)
        if acc == nil {
                return 0
        }
        return acc.Nonce
}

// AddBalance adds amount to an account's balance.
func (s *StateDB) AddBalance(addr Address, amount *big.Int) error {
        if s.GetAccount(addr) == nil {
                if err := s.CreateAccount(addr); err != nil {
                        return err
                }
        }
        return s.accounts[addr].AddBalance(amount)
}

// SubBalance subtracts amount from an account's balance.
func (s *StateDB) SubBalance(addr Address, amount *big.Int) error {
        if s.GetAccount(addr) == nil {
                if err := s.CreateAccount(addr); err != nil {
                        return err
                }
        }
        return s.accounts[addr].SubBalance(amount)
}

// IncrementNonce increments an account's nonce.
func (s *StateDB) IncrementNonce(addr Address) error {
        if s.GetAccount(addr) == nil {
                if err := s.CreateAccount(addr); err != nil {
                        return err
                }
        }
        return s.accounts[addr].IncrementNonce()
}

// Mint increases account balance. Used by genesis/initRewards.
func (s *StateDB) Mint(addr Address, amount *big.Int) error {
        return s.AddBalance(addr, amount)
}

// StateRoot computes the state root hash from all accounts.
func (s *StateDB) StateRoot() Hash {
        h := sha256.New()
        for _, acc := range s.accounts {
                if acc != nil {
                        accHash := acc.Hash()
                        h.Write(accHash[:])
                }
        }
        var out Hash
        copy(out[:], h.Sum(nil))
        return out
}

// Snapshot creates a snapshot of the current state.
func (s *StateDB) Snapshot() int {
        snap := make(map[Address]*Account)
        for addr, acc := range s.accounts {
                snap[addr] = acc.Copy()
        }
        s.snapshots = append(s.snapshots, snap)
        return len(s.snapshots) - 1
}

// RevertToSnapshot reverts the state to a previous snapshot.
func (s *StateDB) RevertToSnapshot(snapID int) {
        if snapID < 0 || snapID >= len(s.snapshots) {
                return
        }
        s.accounts = s.snapshots[snapID]
        s.snapshots = s.snapshots[:snapID]
}

// CommitSnapshot removes a snapshot after successful execution.
func (s *StateDB) CommitSnapshot(snapID int) {
        if snapID < 0 || snapID >= len(s.snapshots) {
                return
        }
        s.snapshots = s.snapshots[:snapID]
}