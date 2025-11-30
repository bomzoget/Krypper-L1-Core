// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"crypto/ecdsa"
	"errors"
	"math/big"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// GenerateKey creates a new ECDSA private key and its corresponding address.
func GenerateKey() (*ecdsa.PrivateKey, Address, error) {
	priv, err := gethcrypto.GenerateKey()
	if err != nil {
		return nil, Address{}, err
	}
	addr := PubKeyToAddress(&priv.PublicKey)
	return priv, addr, nil
}

// PubKeyToAddress derives a KRYPPER Address from an ECDSA public key.
func PubKeyToAddress(pub *ecdsa.PublicKey) Address {
	ethAddr := gethcrypto.PubkeyToAddress(*pub) // 20 bytes
	var addr Address
	copy(addr[:], ethAddr.Bytes())
	return addr
}

// PrivateKeyToAddress derives an address directly from a private key.
func PrivateKeyToAddress(priv *ecdsa.PrivateKey) Address {
	return PubKeyToAddress(&priv.PublicKey)
}

// SignTransaction signs the transaction with the given private key.
// It fills tx.Signature and caches tx.from.
func SignTransaction(tx *Transaction, priv *ecdsa.PrivateKey) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	if priv == nil {
		return errors.New("nil private key")
	}

	// Basic stateless validation first.
	if err := tx.ValidateBasic(); err != nil {
		return err
	}

	// Hash payload (includes ChainID, type, nonce, value, gas, data)
	payload := tx.HashForSign()

	sig, err := gethcrypto.Sign(payload[:], priv)
	if err != nil {
		return err
	}
	if len(sig) != 65 {
		return errors.New("invalid signature length")
	}

	r := new(big.Int).SetBytes(sig[0:32])
	s := new(big.Int).SetBytes(sig[32:64])
	v := uint8(sig[64])

	tx.Signature.R = r
	tx.Signature.S = s
	tx.Signature.V = v

	// Reset cached tx hash since signature changed.
	tx.hash = Hash{}

	// Cache sender address from private key.
	from := PubKeyToAddress(&priv.PublicKey)
	tx.from = &from

	return nil
}

// RecoverTxSender recovers the sender address from the transaction signature.
// It also caches tx.from if recovery succeeds.
func RecoverTxSender(tx *Transaction) (Address, error) {
	if tx == nil {
		return Address{}, errors.New("nil transaction")
	}
	if tx.Signature.R == nil || tx.Signature.S == nil {
		return Address{}, errors.New("missing signature components")
	}

	// Rebuild 65-byte signature from R, S, V.
	sig := make([]byte, 65)

	rBytes := padTo32(tx.Signature.R.Bytes())
	sBytes := padTo32(tx.Signature.S.Bytes())

	copy(sig[0:32], rBytes)
	copy(sig[32:64], sBytes)
	sig[64] = byte(tx.Signature.V)

	// Hash payload exactly as during signing.
	payload := tx.HashForSign()

	pubKey, err := gethcrypto.SigToPub(payload[:], sig)
	if err != nil {
		return Address{}, err
	}

	addr := PubKeyToAddress(pubKey)
	tx.from = &addr

	return addr, nil
}

// VerifyTxSignature verifies that the transaction signature is valid.
// If tx.from is already set, it ensures the recovered address matches it.
func VerifyTxSignature(tx *Transaction) (bool, error) {
	if tx == nil {
		return false, errors.New("nil transaction")
	}

	recovered, err := RecoverTxSender(tx)
	if err != nil {
		return false, err
	}

	expected := tx.GetFrom()
	if expected.IsZero() {
		// No expected sender set, accept recovered as authoritative.
		tx.from = &recovered
		return true, nil
	}

	if recovered != expected {
		return false, errors.New("signature does not match sender")
	}
	return true, nil
}

// padTo32 left-pads the given byte slice to 32 bytes.
func padTo32(b []byte) []byte {
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}