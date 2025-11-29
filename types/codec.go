// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"encoding/json"
	"errors"
)

// EncodeTx serializes a transaction to bytes (JSON-based for now).
func EncodeTx(tx *Transaction) ([]byte, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}
	return json.Marshal(tx)
}

// DecodeTx deserializes a transaction from bytes.
func DecodeTx(data []byte) (*Transaction, error) {
	if len(data) == 0 {
		return nil, errors.New("empty transaction data")
	}
	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

// EncodeBlock serializes a block to bytes.
func EncodeBlock(b *Block) ([]byte, error) {
	if b == nil {
		return nil, errors.New("nil block")
	}
	return json.Marshal(b)
}

// DecodeBlock deserializes a block from bytes.
func DecodeBlock(data []byte) (*Block, error) {
	if len(data) == 0 {
		return nil, errors.New("empty block data")
	}
	var blk Block
	if err := json.Unmarshal(data, &blk); err != nil {
		return nil, err
	}
	return &blk, nil
}