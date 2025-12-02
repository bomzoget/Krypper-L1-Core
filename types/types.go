// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
        "encoding/hex"
)

// =========================
// Hash type (32 bytes)
// =========================

type Hash [32]byte

func (h Hash) String() string {
        return "0x" + hex.EncodeToString(h[:])
}

func (h Hash) IsZero() bool {
        return h == Hash{}
}

func ZeroHash() Hash {
        return Hash{}
}

// Address.IsZero checks if address is zero address
func (a Address) IsZero() bool {
        return a == Address{}
}