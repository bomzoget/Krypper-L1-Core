// SPDX-License-Identifier: MIT
// Dev KryperAI

package types

import (
	"errors"
	"math/big"
)

type Receipt struct {
	TxHash  Hash     `json:"txHash"`
	Success bool     `json:"success"`
	GasUsed uint64   `json:"gasUsed"`
	Logs    [][]byte `json:"logs"`
}

type ChainConfig struct {
	ChainID        uint64
	Coinbase       Address  // miner for this block
	RewardPoolAddr Address  // buyback/reserve
	PoolShare      uint64   // % of gas sent to pool (0-100)
}

type Executor struct {
	state  *StateDB
	config ChainConfig
}

func NewExecutor(state *StateDB, config ChainConfig) *Executor {
	return &Executor{state: state, config: config}
}

func (e *Executor) ExecuteBlock(b *Block) ([]*Receipt, error) {
	if b == nil {
		return nil, errors.New("nil block")
	}

	receipts := make([]*Receipt, 0, len(b.Transactions))

	for _, raw := range b.Transactions {
		tx, err := DecodeTx(raw)
		if err != nil {
			return receipts, err
		}
		if err := tx.ValidateBasic(); err != nil {
			return receipts, err
		}
		r, err := e.ExecuteTx(tx)
		if err != nil {
			return receipts, err
		}
		receipts = append(receipts, r)
	}

	b.Header.StateRoot = e.state.StateRoot()
	return receipts, nil
}

func (e *Executor) ExecuteTx(tx *Transaction) (*Receipt, error) {
	from := tx.GetFrom()
	if from.IsZero() {
		return nil, errors.New("missing sender signature")
	}

	snap := e.state.Snapshot()

	gasFee := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit), tx.GasPrice)
	totalCost := new(big.Int).Add(tx.Value, gasFee)

	if err := e.state.SubBalance(from, totalCost); err != nil {
		e.state.RevertToSnapshot(snap)
		return nil, err
	}

	if err := e.state.IncrementNonce(from); err != nil {
		e.state.RevertToSnapshot(snap)
		return nil, err
	}

	if tx.Value.Sign() > 0 {
		if err := e.state.AddBalance(tx.To, tx.Value); err != nil {
			e.state.RevertToSnapshot(snap)
			return nil, err
		}
	}

	poolAmt := new(big.Int).Mul(gasFee, new(big.Int).SetUint64(e.config.PoolShare))
	poolAmt.Div(poolAmt, big.NewInt(100))
	minerAmt := new(big.Int).Sub(gasFee, poolAmt)

	if poolAmt.Sign() > 0 {
		e.state.AddBalance(e.config.RewardPoolAddr, poolAmt)
	}
	if minerAmt.Sign() > 0 {
		e.state.AddBalance(e.config.Coinbase, minerAmt)
	}

	return &Receipt{
		TxHash:  tx.Hash(),
		Success: true,
		GasUsed: tx.GasLimit,
		Logs:    [][]byte{},
	}, nil
}