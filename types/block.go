// SPDX-License-Identifier: MIT
// Dev KryperAI
package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

type Hash [32]byte

func (h Hash) String() string { return "0x" + hex.EncodeToString(h[:]) }
func ZeroHash() Hash          { return Hash{} }

type Address [20]byte

func (a Address) String() string { return "0x" + hex.EncodeToString(a[:]) }
func (a Address) IsZero() bool   { return a == Address{} }

type BlockHeight uint64

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

type Block struct {
	Header       BlockHeader `json:"header"`
	Transactions [][]byte    `json:"txs"`
}

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

	out := Hash{}
	copy(out[:], hasher.Sum(nil))
	return out
}

func (b *Block) ID() Hash {
	return b.Header.HashHeader()
}

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

	return nil
}