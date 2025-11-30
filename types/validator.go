// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

// ValidatorVote represents a Tier-2 validator attestation for a block.
type ValidatorVote struct {
	ChainID   uint64  `json:"chainId"`
	Height    uint64  `json:"height"`
	BlockHash Hash    `json:"blockHash"`
	Validator Address `json:"validator"`
	Signature []byte  `json:"signature"`
}

// String returns a short debug string for the vote.
func (v *ValidatorVote) String() string {
	return "ValidatorVote{height=" + uintToString(v.Height) +
		", hash=" + v.BlockHash.String() +
		", validator=" + v.Validator.String() + "}"
}

// SigningHash builds the canonical hash that is actually signed by the validator.
func (v *ValidatorVote) SigningHash() Hash {
	var (
		buf   [8]byte
		h     = sha256.New()
		zero  Hash
		out   Hash
	)

	// ChainID
	binary.BigEndian.PutUint64(buf[:], v.ChainID)
	h.Write(buf[:])

	// Height
	binary.BigEndian.PutUint64(buf[:], v.Height)
	h.Write(buf[:])

	// Block hash
	if v.BlockHash == zero {
		// still write 32 zero bytes to keep format stable
		h.Write(zero[:])
	} else {
		h.Write(v.BlockHash[:])
	}

	// Validator address
	h.Write(v.Validator[:])

	sum := h.Sum(nil)
	copy(out[:], sum)
	return out
}

// SignValidatorVote creates and signs a new ValidatorVote.
func SignValidatorVote(priv *ecdsa.PrivateKey, chainID, height uint64, blockHash Hash) (*ValidatorVote, error) {
	if priv == nil {
		return nil, errors.New("nil private key")
	}

	validatorAddr := PubKeyToAddress(&priv.PublicKey)

	vote := &ValidatorVote{
		ChainID:   chainID,
		Height:    height,
		BlockHash: blockHash,
		Validator: validatorAddr,
	}

	hash := vote.SigningHash()

	sig, err := signHashSECP(priv, hash)
	if err != nil {
		return nil, err
	}
	vote.Signature = sig
	return vote, nil
}

// VerifyValidatorVote verifies the signature and returns the recovered address.
func VerifyValidatorVote(vote *ValidatorVote) (Address, error) {
	var zeroAddr Address

	if vote == nil {
		return zeroAddr, errors.New("nil vote")
	}
	if len(vote.Signature) == 0 {
		return zeroAddr, errors.New("empty signature")
	}

	hash := vote.SigningHash()

	addr, err := recoverAddressFromSig(hash, vote.Signature)
	if err != nil {
		return zeroAddr, err
	}

	if addr != vote.Validator {
		return zeroAddr, errors.New("validator mismatch")
	}

	return addr, nil
}

// signHashSECP signs a 32-byte hash using secp256k1 and returns 65-byte signature (R||S||V).
func signHashSECP(priv *ecdsa.PrivateKey, hash Hash) ([]byte, error) {
	sig, err := crypto.Sign(hash[:], priv)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// recoverAddressFromSig recovers the address from hash and signature.
func recoverAddressFromSig(hash Hash, sig []byte) (Address, error) {
	var zeroAddr Address

	if len(sig) != 65 {
		return zeroAddr, errors.New("invalid signature length")
	}

	pubKey, err := crypto.SigToPub(hash[:], sig)
	if err != nil {
		return zeroAddr, err
	}

	addr := PubKeyToAddress(pubKey)
	return addr, nil
}

// Helper: Uint to string without importing strconv in this file.
func uintToString(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [32]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// DebugHex returns a hex string for the vote signature (optional helper).
func (v *ValidatorVote) SigHex() string {
	return "0x" + hex.EncodeToString(v.Signature)
}