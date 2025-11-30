// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

import (
	"math/big"
)

// StateDB is the chain global state container.
// In final implementation this should connect to a persistent DB (LevelDB/MPT),
// but for genesis & bring-up it works fully in-memory.
type StateDB struct {
	accounts map[Address]*Account
}

type Account struct {
	Address Address
	Balance *big.Int
	Stake   *big.Int
	Nonce   uint64
}

func NewStateDB() *StateDB {
	return &StateDB{
		accounts: make(map[Address]*Account),
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
	s.accounts[addr] = &Account{
		Address: addr,
		Balance: big.NewInt(0),
		Stake:   big.NewInt(0),
		Nonce:   0,
	}
	return nil
}

// Mint increases account balance. Used by genesis/initRewards.
func (s *StateDB) Mint(addr Address, amount *big.Int) error {
	if s.GetAccount(addr) == nil {
		if err := s.CreateAccount(addr); err != nil {
			return err
		}
	}
	s.accounts[addr].Balance.Add(s.accounts[addr].Balance, amount)
	return nil
}

// SetStake registers validator stake. Called from genesis.
func (s *StateDB) SetStake(addr Address, stake *big.Int) error {
	if s.GetAccount(addr) == nil {
		if err := s.CreateAccount(addr); err != nil {
			return err
		}
	}
	s.accounts[addr].Stake = new(big.Int).Set(stake)
	return nil
}