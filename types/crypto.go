// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// Wallet represents a local keypair and derived address.
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Address    Address
}

// NewWallet generates a new secp256k1 keypair.
func NewWallet() (*Wallet, error) {
	priv, err := ethcrypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	addr := pubKeyToAddress(&priv.PublicKey)
	return &Wallet{
		PrivateKey: priv,
		Address:    addr,
	}, nil
}

// PrivateKeyToHex exports the private key as hex string (without 0x prefix).
func PrivateKeyToHex(priv *ecdsa.PrivateKey) (string, error) {
	if priv == nil {
		return "", errors.New("nil private key")
	}
	bytes := ethcrypto.FromECDSA(priv)
	return hex.EncodeToString(bytes), nil
}

// PrivateKeyFromHex parses a hex-encoded private key.
func PrivateKeyFromHex(hexKey string) (*ecdsa.PrivateKey, error) {
	if hexKey == "" {
		return nil, errors.New("empty key string")
	}
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	return ethcrypto.ToECDSA(bytes)
}

// pubKeyToAddress derives a KRYPPER-style Address from an ECDSA public key.
// Same scheme as Ethereum: last 20 bytes of keccak256(uncompressed pubkey[1:]).
func pubKeyToAddress(pub *ecdsa.PublicKey) Address {
	pubBytes := ethcrypto.FromECDSAPub(pub) // 65 bytes: 0x04 || X || Y
	hash := ethcrypto.Keccak256(pubBytes[1:])
	var addr Address
	copy(addr[:], hash[12:])
	return addr
}

// SignTransaction signs the transaction payload with the given private key.
// It fills tx.Signature and tx.from, and clears cached hash.
func SignTransaction(tx *Transaction, priv *ecdsa.PrivateKey) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	if priv == nil {
		return errors.New("nil private key")
	}

	h := tx.HashForSign()

	sig, err := ethcrypto.Sign(h[:], priv)
	if err != nil {
		return err
	}
	if len(sig) != 65 {
		return errors.New("unexpected signature length")
	}

	r := new(big.Int).SetBytes(sig[0:32])
	s := new(big.Int).SetBytes(sig[32:64])
	v := uint8(sig[64])

	tx.Signature.R = r
	tx.Signature.S = s
	tx.Signature.V = v

	addr := pubKeyToAddress(&priv.PublicKey)
	tx.SetFrom(addr)
	tx.hash = Hash{} // clear cached hash

	return nil
}

// RecoverTxSender recovers the sender address from the transaction signature.
// It also caches the result in tx.from.
func RecoverTxSender(tx *Transaction) (Address, error) {
	if tx == nil {
		return Address{}, errors.New("nil transaction")
	}
	if tx.Signature.R == nil || tx.Signature.S == nil {
		return Address{}, errors.New("missing signature components")
	}

	h := tx.HashForSign()

	sig := make([]byte, 65)
	rBytes := tx.Signature.R.Bytes()
	sBytes := tx.Signature.S.Bytes()

	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)
	sig[64] = byte(tx.Signature.V)

	pubKey, err := ethcrypto.SigToPub(h[:], sig)
	if err != nil {
		return Address{}, err
	}

	addr := pubKeyToAddress(pubKey)
	tx.SetFrom(addr)

	return addr, nil
}

// VerifyTxSignature checks that the transaction signature is valid
// for its sign-hash payload.
func VerifyTxSignature(tx *Transaction) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	if tx.Signature.R == nil || tx.Signature.S == nil {
		return errors.New("missing signature components")
	}

	h := tx.HashForSign()

	sig := make([]byte, 65)
	rBytes := tx.Signature.R.Bytes()
	sBytes := tx.Signature.S.Bytes()

	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)
	sig[64] = byte(tx.Signature.V)

	pubKey, err := ethcrypto.SigToPub(h[:], sig)
	if err != nil {
		return err
	}

	pubBytes := ethcrypto.FromECDSAPub(pubKey)
	if !ethcrypto.VerifySignature(pubBytes[1:], h[:], sig[:64]) {
		return errors.New("invalid transaction signature")
	}

	addr := pubKeyToAddress(pubKey)
	tx.SetFrom(addr)

	return nil
}