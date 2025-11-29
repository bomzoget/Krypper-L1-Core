// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"crypto/sha256"
	"encoding/binary"
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

// =========================
// Address (20 bytes, EVM-compatible)
// =========================

type Address [20]byte

func (a Address) String() string {
	return "0x" + hex.EncodeToString(a[:])
}

func (a Address) IsZero() bool {
	return a == Address{}
}

// =========================
// Block height
// =========================

type BlockHeight uint64

// =========================
// Block header
// =========================

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

// HashHeader computes the canonical hash of the block header.
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

	binary.BigEndian.PutUint64(buf[:], uint64(len(h.Extra)))
	hasher.Write(buf[:])
	if len(h.Extra) > 0 {
		hasher.Write(h.Extra)
	}

	sum := hasher.Sum(nil)

	var out Hash
	copy(out[:], sum)
	return out
}

// =========================
// Block (header + txs)
// =========================

type Block struct {
	Header       *BlockHeader   `json:"header"`
	Transactions []*Transaction `json:"transactions"`

	hash Hash `json:"-"`
}

// NewBlock constructs a block and binds TxRoot to the header.
func NewBlock(header *BlockHeader, txs []*Transaction) *Block {
	if header == nil {
		header = &BlockHeader{}
	}
	header.TxRoot = CalculateTxRoot(txs)

	return &Block{
		Header:       header,
		Transactions: txs,
	}
}

// Hash returns the block identifier (header hash).
func (b *Block) Hash() Hash {
	if !b.hash.IsZero() {
		return b.hash
	}
	if b.Header == nil {
		return ZeroHash()
	}
	b.hash = b.Header.HashHeader()
	return b.hash
}

// =========================
// Transaction root (Merkle)
// =========================

func CalculateTxRoot(txs []*Transaction) Hash {
	if len(txs) == 0 {
		return ZeroHash()
	}

	leaves := make([]Hash, 0, len(txs))
	for _, tx := range txs {
		if tx == nil {
			continue
		}
		leaves = append(leaves, tx.Hash())
	}

	if len(leaves) == 0 {
		return ZeroHash()
	}

	return buildTxMerkleRoot(leaves)
}

func buildTxMerkleRoot(nodes []Hash) Hash {
	if len(nodes) == 1 {
		return nodes[0]
	}

	current := make([]Hash, len(nodes))
	copy(current, nodes)

	for len(current) > 1 {
		if len(current)%2 != 0 {
			current = append(current, current[len(current)-1])
		}

		next := make([]Hash, 0, len(current)/2)
		for i := 0; i < len(current); i += 2 {
			sum := sha256.Sum256(append(current[i][:], current[i+1][:]...))
			next = append(next, Hash(sum))
		}
		current = next
	}

	return current[0]
}