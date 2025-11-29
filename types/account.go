// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"
)

/* ========================= *
       ACCOUNT STRUCT
* ========================= */

type Account struct {
	Address     Address  `json:"address"`
	Balance     *big.Int `json:"balance"`
	Nonce       uint64   `json:"nonce"`
	CodeHash    Hash     `json:"codeHash"`
	StorageRoot Hash     `json:"storageRoot"`
	Frozen      bool     `json:"frozen"` // used for slashing / penalties
}

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

/* ========================= *
       ACCOUNT → HASH
* ========================= */

func (a *Account) Hash() Hash {
	hasher := sha256.New()

	// 1) Address — deterministic, fixed length
	hasher.Write(a.Address[:])

	// 2) Balance — include zero cleanly (no empty hash ambiguity)
	if a.Balance != nil && a.Balance.Sign() != 0 {
		hasher.Write(a.Balance.Bytes())
	} else {
		hasher.Write([]byte{0}) // critical fix
	}

	// 3) Nonce — encoded as uint64
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], a.Nonce)
	hasher.Write(buf[:])

	// 4) Smart Contract compatibility fields
	hasher.Write(a.CodeHash[:])
	hasher.Write(a.StorageRoot[:])

	// 5) Security flag
	if a.Frozen {
		hasher.Write([]byte{1})
	} else {
		hasher.Write([]byte{0})
	}

	out := Hash{}
	copy(out[:], hasher.Sum(nil))
	return out
}

/* ========================= *
      MUTATION FUNCTIONS
* ========================= */

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