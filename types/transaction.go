// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"
)

/*
   KRYPPER L1 CORE TRANSACTION MODEL (FINAL)
   - Anti replay (ChainId included in sign-hash)
   - From is recovered from signature (not stored)
   - big.Int length-prefixed → no ambiguity
   - Cached TxID for performance
*/

type TxType uint8

const (
	TxTypeTransfer TxType = 0x01
)

type Signature struct {
	R *big.Int `json:"r"`
	S *big.Int `json:"s"`
	V uint8    `json:"v"`
}

type Transaction struct {
	ChainId   *big.Int  `json:"chainId"` // 🔥 REQUIRED to prevent replay
	Type      TxType    `json:"type"`
	Nonce     uint64    `json:"nonce"`
	To        Address   `json:"to"`
	Value     *big.Int  `json:"value"`
	GasPrice  *big.Int  `json:"gasPrice"`
	GasLimit  uint64    `json:"gasLimit"`
	Data      []byte    `json:"data"`
	Signature Signature `json:"sig"`

	from *Address `json:"-"` // derived after signature recovery
	hash Hash     `json:"-"` // cached TxID
}

/* Constructor (FIXED) */
func NewTransferTx(
	chainId uint64,
	nonce uint64,
	to Address,
	value, gasPrice *big.Int,
	gasLimit uint64,
	data []byte,
) *Transaction {
	if value == nil {
		value = big.NewInt(0)
	}
	if gasPrice == nil {
		gasPrice = big.NewInt(0)
	}

	return &Transaction{
		ChainId:  new(big.Int).SetUint64(chainId), // 🔥 FIXED (no overflow, no negative wrap)
		Type:     TxTypeTransfer,
		Nonce:    nonce,
		To:       to,
		Value:    new(big.Int).Set(value),
		GasPrice: new(big.Int).Set(gasPrice),
		GasLimit: gasLimit,
		Data:     data,
		Signature: Signature{
			R: big.NewInt(0), S: big.NewInt(0), V: 0,
		},
	}
}

/* ------------------------------------------------------- *
   SIGN-HASH (What user signs)
   * MUST include ChainID = anti replay
* ------------------------------------------------------- */
func (tx *Transaction) HashForSign() Hash {
	h := sha256.New()
	var buf [8]byte

	// 1) Chain ID first — critical
	writeBig(h, tx.ChainId)

	// 2) Tx fields
	h.Write([]byte{byte(tx.Type)})

	binary.BigEndian.PutUint64(buf[:], tx.Nonce)
	h.Write(buf[:])

	h.Write(tx.To[:])

	writeBig(h, tx.Value)
	writeBig(h, tx.GasPrice)

	binary.BigEndian.PutUint64(buf[:], tx.GasLimit)
	h.Write(buf[:])

	binary.BigEndian.PutUint64(buf[:], uint64(len(tx.Data)))
	h.Write(buf[:])
	if len(tx.Data) > 0 {
		h.Write(tx.Data)
	}

	var out Hash
	copy(out[:], h.Sum(nil))
	return out
}

/* ------------------------------------------------------- *
   TxID = Hash(payload + signature)
* ------------------------------------------------------- */
func (tx *Transaction) Hash() Hash {
	if !tx.hash.IsZero() {
		return tx.hash
	}

	h := sha256.New()
	payload := tx.HashForSign()
	h.Write(payload[:])

	writeBig(h, tx.Signature.R)
	writeBig(h, tx.Signature.S)
	h.Write([]byte{tx.Signature.V})

	copy(tx.hash[:], h.Sum(nil))
	return tx.hash
}

/* Helper — Secure big serialization */
func writeBig(w interface{ Write([]byte) (int, error) }, n *big.Int) {
	if n == nil || n.Sign() == 0 {
		w.Write([]byte{0})
		return
	}
	b := n.Bytes()
	w.Write([]byte{uint8(len(b))})
	w.Write(b)
}

/* ------------------------------------------------------- *
   Basic validation
* ------------------------------------------------------- */
func (tx *Transaction) ValidateBasic() error {
	if tx == nil {
		return errors.New("nil tx")
	}
	if tx.ChainId == nil || tx.ChainId.Sign() <= 0 {
		return errors.New("invalid chainId")
	}
	if tx.Type != TxTypeTransfer {
		return errors.New("unsupported tx type")
	}
	if tx.Value == nil || tx.Value.Sign() < 0 {
		return errors.New("invalid value")
	}
	if tx.GasLimit == 0 {
		return errors.New("gasLimit must > 0")
	}
	if tx.GasPrice == nil || tx.GasPrice.Sign() < 0 {
		return errors.New("invalid gas price")
	}
	return nil
}

func (tx *Transaction) SetFrom(a Address) { tx.from = &a }
func (tx *Transaction) GetFrom() Address {
	if tx.from == nil {
		return Address{}
	}
	return *tx.from
}

func (tx *Transaction) String() string { return "Tx{" + hex.EncodeToString(tx.Hash()[:]) + "}" }