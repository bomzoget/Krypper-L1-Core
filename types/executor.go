// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

import (
        "errors"
        "math/big"
)

type Receipt struct {
        TxHash  Hash
        Success bool
        GasUsed uint64
        Logs    [][]byte
}

// Tier-based reward config
type ChainConfig struct {
        ChainID    uint64
        RewardPool Address

        ShareTier1 uint64 // Proposer
        ShareTier2 uint64 // Validator
        ShareTier3 uint64 // Witness
        SharePool  uint64 // Reserve/Fund

        // % total <= 100 → remainder = auto-burn
}

type Executor struct {
        state   *StateDB
        config  ChainConfig
        current *BlockHeader
}

func NewExecutor(state *StateDB, cfg ChainConfig) *Executor {
        return &Executor{state: state, config: cfg}
}

func (e *Executor) SetBlock(h *BlockHeader) { e.current = h }

func (e *Executor) SetCurrentHeader(h *BlockHeader) { e.current = h }

func (e *Executor) SetCoinbase(addr Address) {
        if e.current != nil {
                e.current.Proposer = addr
        }
}

// -------------------------------------------------------------

func (e *Executor) ExecuteBlock(b *Block) ([]*Receipt, error) {
        if b == nil || b.Header == nil {
                return nil, errors.New("invalid block")
        }

        e.current = b.Header
        receipts := make([]*Receipt, len(b.Transactions))

        for i, tx := range b.Transactions {
                r, err := e.ExecuteTx(tx)
                if err != nil {
                        return receipts[:i], err
                }
                receipts[i] = r
        }

        return receipts, nil
}

// -------------------------------------------------------------

func (e *Executor) ExecuteTx(tx *Transaction) (*Receipt, error) {
        if tx == nil {
                return nil, errors.New("nil tx")
        }

        from, err := RecoverTxSender(tx)
        if err != nil {
                return nil, errors.New("invalid signature")
        }

        snap := e.state.Snapshot() // <- rollback layer

        fee := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit), tx.GasPrice)
        total := new(big.Int).Add(tx.Value, fee)

        if err := e.state.SubBalance(from, total); err != nil {
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

        // ---------------------------------------------------------
        // 🔥 Tier reward distribution
        // ---------------------------------------------------------
        t1 := calcPct(fee, e.config.ShareTier1)
        t2 := calcPct(fee, e.config.ShareTier2)
        t3 := calcPct(fee, e.config.ShareTier3)
        pfund := calcPct(fee, e.config.SharePool)

        if t1.Sign() > 0 && !e.current.Proposer.IsZero() {
                e.state.AddBalance(e.current.Proposer, t1)
        }
        if t2.Sign() > 0 && !e.current.Validator.IsZero() {
                e.state.AddBalance(e.current.Validator, t2)
        }
        if t3.Sign() > 0 && !e.current.Witness.IsZero() {
                e.state.AddBalance(e.current.Witness, t3)
        }
        if pfund.Sign() > 0 {
                e.state.AddBalance(e.config.RewardPool, pfund)
        }

        // ---------------------------------------------------------
        // 🧹 Important fix → clear snapshot (prevent RAM leak)
        // ---------------------------------------------------------
        e.state.CommitSnapshot(snap)

        return &Receipt{
                TxHash:  tx.Hash(),
                Success: true,
                GasUsed: tx.GasLimit,
                Logs:    nil,
        }, nil
}

func calcPct(base *big.Int, pct uint64) *big.Int {
        if pct == 0 {
                return big.NewInt(0)
        }
        out := new(big.Int).Mul(base, new(big.Int).SetUint64(pct))
        return out.Div(out, big.NewInt(100))
}