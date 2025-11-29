// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

/* ========================= *
    BASE PRIMITIVES
* ========================= */

type Hash [32]byte

func (h Hash) String() string { return "0x" + hex.EncodeToString(h[:]) }
func ZeroHash() Hash          { return Hash{} }

type Address [20]byte

func (a Address) String() string { return "0x" + hex.EncodeToString(a[:]) }
func (a Address) IsZero() bool   { return a == Address{} }

type BlockHeight uint64

/* ========================= *
          BLOCK HEADER
* ========================= */

type BlockHeader struct {
	ParentHash   Hash        `json:"parentHash"`
	StateRoot    Hash        `json:"stateRoot"`
	TxRoot       Hash        `json:"txRoot"`
	ReceiptsRoot Hash        `json:"receiptsRoot"`
	Height       BlockHeight `json:"height"`
	Timestamp    int64       `json:"timestamp"`
	GasUsed      uint64      `json:"gasUsed"`
	GasLimit     uint64      `json:"gasLimit"`
	Proposer     Address     `json:"proposer"`
	Extra        []byte      `json:"extra"`
}

/* ========================= *
           BLOCK BODY
* ========================= */

type Block struct {
	Header       BlockHeader `json:"header"`
	Transactions [][]byte    `json:"txs"`
}

/* ========================= *
      MERKLE IMPLEMENTATION
* ========================= */

func MerkleRoot(txs [][]byte) Hash {
	if len(txs) == 0 {
		return ZeroHash()
	}

	// Hash each transaction → leaf
	var hashes []Hash
	for _, tx := range txs {
		sum := sha256.Sum256(tx)
		hashes = append(hashes, sum)
	}

	// Build trees upward
	for len(hashes) > 1 {
		// odd → duplicate last
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		var next []Hash
		for i := 0; i < len(hashes); i += 2 {
			h := sha256.Sum256(append(hashes[i][:], hashes[i+1][:]...))
			next = append(next, h)
		}
		hashes = next
	}

	return hashes[0]
}

// Called only when BUILDING blocks
func (b *Block) ComputeRoots() {
	b.Header.TxRoot = MerkleRoot(b.Transactions)
}

/* ========================= *
      HEADER HASH = BLOCK ID
* ========================= */

func (h *BlockHeader) HashHeader() Hash {
	hasher := sha256.New()

	hasher.Write(h.ParentHash[:])
	hasher.Write(h.StateRoot[:])
	hasher.Write(h.TxRoot[:])
	hasher.Write(h.ReceiptsRoot[:])

	var buf [8]byte

	binary.BigEndian.PutUint64(buf[:], uint64(h.Height))
	hasher.Write(buf[:])

	binary.BigEndian.PutUint64(buf[:], uint64(h.Timestamp))
	hasher.Write(buf[:])

	binary.BigEndian.PutUint64(buf[:], h.GasUsed)
	hasher.Write(buf[:])

	binary.BigEndian.PutUint64(buf[:], h.GasLimit)
	hasher.Write(buf[:])

	hasher.Write(h.Proposer[:])

	// extra must be length-prefix for security
	binary.BigEndian.PutUint64(buf[:], uint64(len(h.Extra)))
	hasher.Write(buf[:])

	if len(h.Extra) > 0 {
		hasher.Write(h.Extra)
	}

	out := Hash{}
	copy(out[:], hasher.Sum(nil))
	return out
}

// ⛔ no mutation inside — read only = SAFE
func (b *Block) ID() Hash {
	return b.Header.HashHeader()
}

/* ========================= *
       VALIDATION RULES
* ========================= */

func (b *Block) ValidateBasic() error {
	if b == nil {
		return errors.New("nil block")
	}

	h := b.Header

	if h.GasLimit < 5000 {
		return errors.New("gasLimit too low")
	}
	if h.GasUsed > h.GasLimit {
		return errors.New("gasUsed > gasLimit")
	}
	if h.Timestamp <= 0 {
		return errors.New("invalid timestamp")
	}
	if h.Height > 0 && h.Proposer.IsZero() {
		return errors.New("zero proposer on non-genesis")
	}
	if len(h.Extra) > 1024 {
		return errors.New("extra too large")
	}

	// verify TxRoot authenticity
	if h.TxRoot != MerkleRoot(b.Transactions) {
		return errors.New("invalid merkle root: tx mismatch")
	}

	return nil
}