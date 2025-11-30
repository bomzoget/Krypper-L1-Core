// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"errors"
	"math/big"
)

// Receipt describes the result of executing a single transaction.
type Receipt struct {
	TxHash  Hash     `json:"txHash"`
	Success bool     `json:"success"`
	GasUsed uint64   `json:"gasUsed"`
	Logs    [][]byte `json:"logs"`
}

// ChainConfig carries execution-time parameters such as fee routing.
type ChainConfig struct {
	ChainID     uint64  // network chain id, informational here
	Coinbase    Address // current block producer / miner
	RewardPool  Address // reserve / buyback pool
	PoolShare   uint64  // percent of gas fee sent to RewardPool (0-100)
}

// Executor applies transactions to StateDB under a given config.
type Executor struct {
	state  *StateDB
	config ChainConfig
}

// NewExecutor constructs an executor bound to a statedb and config.
func NewExecutor(state *StateDB, cfg ChainConfig) *Executor {
	return &Executor{
		state:  state,
		config: cfg,
	}
}

// SetCoinbase updates the coinbase address (for new block producer).
func (e *Executor) SetCoinbase(addr Address) {
	e.config.Coinbase = addr
}

// ExecuteBlock executes all transactions in a block sequentially.
// It assumes the caller (Blockchain) wraps this with a higher-level
// snapshot/revert for the entire block if needed.
func (e *Executor) ExecuteBlock(b *Block) ([]*Receipt, error) {
	if b == nil {
		return nil, errors.New("nil block")
	}

	receipts := make([]*Receipt, 0, len(b.Transactions))

	for _, tx := range b.Transactions {
		if tx == nil {
			return receipts, errors.New("nil transaction in block")
		}

		// Stateless validation (already covered in Block.ValidateBasic but harmless).
		if err := tx.ValidateBasic(); err != nil {
			return receipts, err
		}

		// Execute single transaction.
		r, err := e.ExecuteTx(tx)
		if err != nil {
			return receipts, err
		}
		receipts = append(receipts, r)
	}

	return receipts, nil
}

// ExecuteTx executes a single transaction atomically.
// It uses StateDB snapshot/revert to ensure all-or-nothing semantics.
func (e *Executor) ExecuteTx(tx *Transaction) (*Receipt, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}

	// Recover or use cached sender.
	from := tx.GetFrom()
	if from.IsZero() {
		addr, err := RecoverTxSender(tx)
		if err != nil {
			return nil, err
		}
		from = addr
	}

	// Check nonce ordering.
	expectedNonce := e.state.GetNonce(from)
	if tx.Nonce != expectedNonce {
		return nil, errors.New("invalid nonce")
	}

	// Validate gas price.
	if tx.GasPrice == nil || tx.GasPrice.Sign() < 0 {
		return nil, errors.New("invalid gasPrice")
	}

	// Compute gas fee = GasLimit * GasPrice.
	gasFee := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit), tx.GasPrice)

	// Total cost = value + gas fee.
	totalCost := new(big.Int).Add(tx.Value, gasFee)

	// Take a snapshot so this tx can be reverted independently.
	snapID := e.state.Snapshot()

	// 1. Deduct total cost from sender.
	if err := e.state.SubBalance(from, totalCost); err != nil {
		e.state.RevertToSnapshot(snapID)
		return nil, err
	}

	// 2. Increment nonce.
	if err := e.state.IncrementNonce(from); err != nil {
		e.state.RevertToSnapshot(snapID)
		return nil, err
	}

	// 3. Transfer value to recipient.
	if tx.Value != nil && tx.Value.Sign() > 0 {
		if err := e.state.AddBalance(tx.To, tx.Value); err != nil {
			e.state.RevertToSnapshot(snapID)
			return nil, err
		}
	}

	// 4. Distribute gas fee between reward pool and coinbase.
	if gasFee.Sign() > 0 {
		if e.config.PoolShare > 100 {
			e.state.RevertToSnapshot(snapID)
			return nil, errors.New("invalid pool share")
		}

		poolAmount := new(big.Int)
		minerAmount := new(big.Int).Set(gasFee)

		if e.config.PoolShare > 0 {
			poolAmount.Mul(gasFee, big.NewInt(int64(e.config.PoolShare)))
			poolAmount.Div(poolAmount, big.NewInt(100))
			minerAmount.Sub(gasFee, poolAmount)
		}

		if poolAmount.Sign() > 0 {
			_ = e.state.AddBalance(e.config.RewardPool, poolAmount)
		}
		if minerAmount.Sign() > 0 {
			_ = e.state.AddBalance(e.config.Coinbase, minerAmount)
		}
	}

	// Commit tx snapshot; state is now permanent for this tx.
	e.state.CommitSnapshot(snapID)

	receipt := &Receipt{
		TxHash:  tx.Hash(),
		Success: true,
		GasUsed: tx.GasLimit, // placeholder: full gas limit; real VM would track actual used gas.
		Logs:    [][]byte{},
	}

	return receipt, nil
}