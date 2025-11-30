// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

// ValidatorVote represents a Tier2 validator attestation for a block.
type ValidatorVote struct {
	ChainID uint64 `json:"chainId"`
	Height  uint64 `json:"height"`
	Block   Hash   `json:"blockHash"`

	Voter Address `json:"voter"`

	R *big.Int `json:"r"`
	S *big.Int `json:"s"`
	V uint8    `json:"v"`
}

func (v *ValidatorVote) hashForSign() Hash {
	h := sha256.New()
	var buf [8]byte

	binary.BigEndian.PutUint64(buf[:], v.ChainID)
	h.Write(buf[:])

	binary.BigEndian.PutUint64(buf[:], v.Height)
	h.Write(buf[:])

	h.Write(v.Block[:])
	h.Write(v.Voter[:])

	var out Hash
	copy(out[:], h.Sum(nil))
	return out
}

// SignValidatorVote builds a vote and signs it with the given private key.
func SignValidatorVote(
	priv *ecdsa.PrivateKey,
	chainID uint64,
	height uint64,
	blockHash Hash,
) (*ValidatorVote, error) {
	if priv == nil {
		return nil, errors.New("nil private key")
	}

	voter := PubKeyToAddress(&priv.PublicKey)

	vote := &ValidatorVote{
		ChainID: chainID,
		Height:  height,
		Block:   blockHash,
		Voter:   voter,
		R:       big.NewInt(0),
		S:       big.NewInt(0),
		V:       0,
	}

	h := vote.hashForSign()

	sig, err := crypto.Sign(h[:], priv)
	if err != nil {
		return nil, err
	}

	if len(sig) != 65 {
		return nil, errors.New("unexpected signature length")
	}

	vote.R = new(big.Int).SetBytes(sig[0:32])
	vote.S = new(big.Int).SetBytes(sig[32:64])
	vote.V = sig[64]

	return vote, nil
}

// VerifyValidatorVote checks signature and returns the recovered voter address.
func VerifyValidatorVote(v *ValidatorVote) (Address, error) {
	var zero Address

	if v == nil {
		return zero, errors.New("nil vote")
	}

	msgHash := v.hashForSign()

	rBytes := v.R.Bytes()
	sBytes := v.S.Bytes()

	rPadded := make([]byte, 32)
	sPadded := make([]byte, 32)
	copy(rPadded[32-len(rBytes):], rBytes)
	copy(sPadded[32-len(sBytes):], sBytes)

	sig := make([]byte, 65)
	copy(sig[0:32], rPadded)
	copy(sig[32:64], sPadded)
	sig[64] = v.V

	pubKey, err := crypto.SigToPub(msgHash[:], sig)
	if err != nil {
		return zero, err
	}

	recovered := PubKeyToAddress(pubKey)
	if recovered != v.Voter {
		return zero, errors.New("voter mismatch")
	}

	return recovered, nil
}