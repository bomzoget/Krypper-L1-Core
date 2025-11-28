package types

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// Hash is a 32-byte value used for block IDs, roots, and generic hashing.
type Hash [32]byte

// Bytes returns a slice copy of the hash.
func (h Hash) Bytes() []byte {
	b := make([]byte, len(h))
	copy(b, h[:])
	return b
}

// String returns the hex representation (0x-prefixed) of the hash.
func (h Hash) String() string {
	const hexChars = "0123456789abcdef"
	out := make([]byte, 2+len(h)*2)
	out[0], out[1] = '0', 'x'
	i := 2
	for _, b := range h {
		out[i] = hexChars[b>>4]
		out[i+1] = hexChars[b&0x0f]
		i += 2
	}
	return string(out)
}

// ZeroHash returns an all-zero hash.
func ZeroHash() Hash {
	return Hash{}
}

// Address is a 20-byte account / validator address.
// NOTE: type will be shared with accounts, validators, rewards.
type Address [20]byte

func (a Address) String() string {
	const hexChars = "0123456789abcdef"
	out := make([]byte, 2+len(a)*2)
	out[0], out[1] = '0', 'x'
	i := 2
	for _, b := range a {
		out[i] = hexChars[b>>4]
		out[i+1] = hexChars[b&0x0f]
		i += 2
	}
	return string(out)
}

// BlockHeight is the chain height (0-based).
type BlockHeight uint64

// BlockHeader contains all metadata that is signed / agreed in consensus.
//
// Important: This struct must remain stable once mainnet is launched.
// Changes must be backward compatible or handled by versioning.
type BlockHeader struct {
	ParentHash   Hash        `json:"parentHash"`   // hash of the parent block
	StateRoot    Hash        `json:"stateRoot"`    // global state commitment
	TxRoot       Hash        `json:"txRoot"`       // merkle root of transactions
	ReceiptsRoot Hash        `json:"receiptsRoot"` // merkle root of receipts (optional for v1)
	Height       BlockHeight `json:"height"`       // block number / height
	Timestamp    int64       `json:"timestamp"`    // unix seconds
	GasUsed      uint64      `json:"gasUsed"`      // total gas used by all txs
	GasLimit     uint64      `json:"gasLimit"`     // max gas allowed in this block

	Proposer Address `json:"proposer"` // Tier1 validator that proposed this block

	// Extra is reserved for future use (e.g. FastBFT metadata, AI-score hints, chain upgrades).
	Extra []byte `json:"extra"`
}

// Block is a full block: header + transaction payloads.
// For core storage, we keep transactions as raw bytes; higher layers
// can decode them into richer structs.
type Block struct {
	Header       BlockHeader `json:"header"`
	Transactions [][]byte    `json:"transactions"`
}

// BlockID is an alias to Hash for semantic clarity.
type BlockID = Hash

// HashHeader computes the canonical hash of a block header.
// This should be stable and deterministic; any consensus logic
// that signs / verifies blocks must rely on this.
func (h *BlockHeader) HashHeader() Hash {
	// Simple canonical encoding using SHA-256 over the ordered fields.
	// If we later switch to a different encoding (e.g. RLP/SSZ/Protobuff),
	// it must preserve this order for compatibility.
	hasher := sha256.New()

	hasher.Write(h.ParentHash[:])
	hasher.Write(h.StateRoot[:])
	hasher.Write(h.TxRoot[:])
	hasher.Write(h.ReceiptsRoot[:])

	var buf8 [8]byte

	binary.BigEndian.PutUint64(buf8[:], uint64(h.Height))
	hasher.Write(buf8[:])

	binary.BigEndian.PutUint64(buf8[:], uint64(h.Timestamp))
	hasher.Write(buf8[:])

	binary.BigEndian.PutUint64(buf8[:], h.GasUsed)
	hasher.Write(buf8[:])

	binary.BigEndian.PutUint64(buf8[:], h.GasLimit)
	hasher.Write(buf8[:])

	hasher.Write(h.Proposer[:])

	// Extra is included raw; callers should keep its semantics stable.
	if len(h.Extra) > 0 {
		hasher.Write(h.Extra)
	}

	sum := hasher.Sum(nil)

	var out Hash
	copy(out[:], sum)
	return out
}

// ID returns the canonical block ID for this block (header hash).
func (b *Block) ID() BlockID {
	return b.Header.HashHeader()
}

// ValidateBasic performs basic stateless sanity checks on the block.
// It does not verify signatures, state transitions, or consensus rules.
// Those belong to higher layers (consensus / execution).
func (b *Block) ValidateBasic() error {
	if b == nil {
		return fmt.Errorf("block is nil")
	}

	h := b.Header

	// Height 0 is allowed (genesis), but a production node can add more rules later.
	// GasLimit must be > 0 once chain is running.
	if h.GasLimit == 0 {
		return fmt.Errorf("gasLimit must be > 0")
	}

	// Timestamp sanity: must be non-negative.
	if h.Timestamp < 0 {
		return fmt.Errorf("timestamp must be non-negative")
	}

	// Proposer must not be zero address (except genesis; higher layers may allow special rules).
	var zeroAddr Address
	if h.Proposer == zeroAddr && h.Height != 0 {
		return fmt.Errorf("proposer must not be zero for non-genesis blocks")
	}

	// We don't enforce any rule about Transactions here; an empty block is valid.
	return nil
}
