// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

import (
        "encoding/hex"
        "errors"
        "strings"

        "github.com/ethereum/go-ethereum/crypto"
)

const AddressLength = 20

// Address is a fixed 20-byte identifier for accounts/validators.
type Address [AddressLength]byte

// String returns 0x prefixed hex form.
func (a Address) String() string {
        return "0x" + hex.EncodeToString(a[:])
}

// ParseAddress converts hex string -> Address format.
func ParseAddress(s string) (Address, error) {
        if strings.HasPrefix(s, "0x") {
                s = s[2:]
        }
        if len(s) != AddressLength*2 {
                return Address{}, errors.New("invalid address length")
        }
        data, err := hex.DecodeString(s)
        if err != nil {
                return Address{}, err
        }
        var addr Address
        copy(addr[:], data)
        return addr, nil
}

// HexToAddress is an alias for ParseAddress.
func HexToAddress(s string) (Address, error) {
        return ParseAddress(s)
}

// AddressFromPrivateKey derives an address from a private key.
func AddressFromPrivateKey(privKeyBytes []byte) (Address, error) {
        privKey, err := crypto.ToECDSA(privKeyBytes)
        if err != nil {
                return Address{}, err
        }
        pubKey := privKey.PublicKey
        ethAddr := crypto.PubkeyToAddress(pubKey)
        var addr Address
        copy(addr[:], ethAddr.Bytes())
        return addr, nil
}
