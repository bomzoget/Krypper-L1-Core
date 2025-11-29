// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"
)

// Account represents a single world-state account.
type Account struct {
	Address     Address  `json:"address"`
	Balance     *big.Int `json:"balance"`
	Nonce       uint64   `json:"nonce"`
	CodeHash    Hash     `json:"codeHash"`
	StorageRoot Hash     `json:"storageRoot"`
	Frozen      bool     `json:"frozen"`
}

// NewAccount initializes a zeroed account for a given address.
func NewAccount(addr Address) *Account {
	return &Account{
		Address:     addr,
		Balance:     big.NewInt(0),
		Nonce:       0,
		CodeHash:    ZeroHash(),
		StorageRoot: ZeroHash(),
		Frozen:      false,
	}
}

// Copy returns a deep copy of the account.
func (a *Account) Copy() *Account {
	if a == nil {
		return nil
	}

	balCopy := big.NewInt(0)
	if a.Balance != nil {
		balCopy.Set(a.Balance)
	}

	return &Account{
		Address:     a.Address,
		Balance:     balCopy,
		Nonce:       a.Nonce,
		CodeHash:    a.CodeHash,
		StorageRoot: a.StorageRoot,
		Frozen:      a.Frozen,
	}
}

func (a *Account) AddBalance(amount *big.Int) error {
	if a == nil {
		return errors.New("nil account")
	}
	if amount == nil || amount.Sign() < 0 {
		return errors.New("amount must be non-negative")
	}
	a.Balance.Add(a.Balance, amount)
	return nil
}

func (a *Account) SubBalance(amount *big.Int) error {
	if a == nil {
		return errors.New("nil account")
	}
	if amount == nil || amount.Sign() < 0 {
		return errors.New("amount must be non-negative")
	}
	if a.Balance.Cmp(amount) < 0 {
		return errors.New("insufficient balance")
	}
	a.Balance.Sub(a.Balance, amount)
	return nil
}

func (a *Account) IncrementNonce() error {
	if a == nil {
		return errors.New("nil account")
	}
	a.Nonce++
	return nil
}

// Hash computes a hash of the account state for use in state roots.
func (a *Account) Hash() Hash {
	h := sha256.New()

	// Address
	h.Write(a.Address[:])

	// Balance (length-prefixed big-int)
	if a.Balance != nil && a.Balance.Sign() != 0 {
		b := a.Balance.Bytes()
		h.Write([]byte{uint8(len(b))})
		h.Write(b)
	} else {
		h.Write([]byte{0})
	}

	// Nonce
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], a.Nonce)
	h.Write(buf[:])

	// Code and storage roots
	h.Write(a.CodeHash[:])
	h.Write(a.StorageRoot[:])

	// Frozen flag
	if a.Frozen {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}

	var out Hash
	copy(out[:], h.Sum(nil))
	return out
}