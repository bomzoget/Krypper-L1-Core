// SPDX-License-Identifier: MIT
// Dev: KryperAI

package node

import (
        "log"
        "sync"
        "time"

        "krypper-chain/types"
)

type Node struct {
        mu sync.RWMutex

        Chain    *types.Blockchain
        State    *types.StateDB
        Mempool  *types.Mempool
        Executor *types.Executor

        MinerAddress types.Address

        // Tier-3 mobile witnesses
        witnessQueue []types.Witness

        // Tier-2 validator votes, keyed by block height
        validatorVotes map[uint64][]types.ValidatorVote

        Running   bool
        BlockTime time.Duration
}

func NewNode(
        chain *types.Blockchain,
        state *types.StateDB,
        mem *types.Mempool,
        exec *types.Executor,
        minerAddr types.Address,
) *Node {
        return &Node{
                Chain:          chain,
                State:          state,
                Mempool:        mem,
                Executor:       exec,
                MinerAddress:   minerAddr,
                BlockTime:      5 * time.Second,
                witnessQueue:   make([]types.Witness, 0),
                validatorVotes: make(map[uint64][]types.ValidatorVote),
        }
}

func (n *Node) Start() {
        n.mu.Lock()
        if n.Running {
                n.mu.Unlock()
                return
        }
        n.Running = true
        n.mu.Unlock()

        log.Println("[node] started, mining loop active")
        go n.miningLoop()
}

func (n *Node) Stop() {
        n.mu.Lock()
        n.Running = false
        n.mu.Unlock()
        log.Println("[node] stopped")
}

func (n *Node) IsRunning() bool {
        n.mu.RLock()
        defer n.mu.RUnlock()
        return n.Running
}

// AddWitness enqueues a Tier-3 witness for the next blocks.
func (n *Node) AddWitness(w types.Witness) {
        n.mu.Lock()
        defer n.mu.Unlock()

        n.witnessQueue = append(n.witnessQueue, w)
}

// AddValidatorVote stores a Tier-2 validator vote for the current head block.
func (n *Node) AddValidatorVote(v types.ValidatorVote) error {
        n.mu.Lock()
        defer n.mu.Unlock()

        // Stateless verify
        _, err := types.VerifyValidatorVote(&v)
        if err != nil {
                return err
        }

        head := n.Chain.Head()
        if head == nil {
                return nil
        }

        // Only accept votes for current head
        if v.Height != head.Header.Height || v.Block != head.Hash() {
                return nil
        }

        list := n.validatorVotes[v.Height]
        // Deduplicate by validator address
        for _, existing := range list {
                if existing.Voter == v.Voter {
                        return nil
                }
        }

        n.validatorVotes[v.Height] = append(list, v)
        return nil
}

// miningLoop periodically attempts to build and commit new blocks from the mempool.
func (n *Node) miningLoop() {
        ticker := time.NewTicker(n.BlockTime)
        defer ticker.Stop()

        for {
                if !n.IsRunning() {
                        return
                }

                <-ticker.C

                // Select transactions from mempool
                txs := n.Mempool.PopForBlock(100)
                if len(txs) == 0 {
                        continue
                }

                if err := n.createAndSubmitBlock(txs); err != nil {
                        log.Printf("[node] mining error: %v\n", err)
                }
        }
}

// createAndSubmitBlock builds a new block with selected txs and submits it to the chain.
func (n *Node) createAndSubmitBlock(txs []*types.Transaction) error {
        n.mu.Lock()
        defer n.mu.Unlock()

        head := n.Chain.Head()
        if head == nil {
                log.Println("[node] cannot mine: no head block")
                return nil
        }

        // --- pick witness (Tier-3) ---
        var witnessAddr types.Address
        if len(n.witnessQueue) > 0 {
                w := n.witnessQueue[0]
                witnessAddr = w.Address
                // remove used witness
                n.witnessQueue = n.witnessQueue[1:]
        }

        // --- pick validator (Tier-2) ---
        var validatorAddr types.Address
        parentHeight := head.Header.Height
        if votes, ok := n.validatorVotes[parentHeight]; ok && len(votes) > 0 {
                // for now: pick the first vote
                validatorAddr = votes[0].Voter
                // clear stored votes for this height to avoid unbounded growth
                delete(n.validatorVotes, parentHeight)
        }

        // build header skeleton
        header := &types.BlockHeader{
                ParentHash: head.Hash(),
                Height:     head.Header.Height + 1,
                Timestamp:  time.Now().Unix(),
                Proposer:   n.MinerAddress,
                Validator:  validatorAddr,
                Witness:    witnessAddr,
                GasLimit:   30_000_000,
        }

        // dry-run execution to compute StateRoot
        snap := n.State.Snapshot()

        // ensure the executor knows which block header is currently being executed
        n.Executor.SetCurrentHeader(header)

        for _, tx := range txs {
                if _, err := n.Executor.ExecuteTx(tx); err != nil {
                        // revert and drop this block attempt
                        n.State.RevertToSnapshot(snap)
                        return err
                }
        }

        header.StateRoot = n.State.StateRoot()

        // revert dry-run; Blockchain.AddBlock will run execution again atomically
        n.State.RevertToSnapshot(snap)

        // finalize block
        block := types.NewBlock(header, txs)
        block.ComputeTxRoot()

        // submit to chain (this will do a real execution + state root check + commit)
        if err := n.Chain.AddBlock(block); err != nil {
                return err
        }

        log.Printf("[node] new block committed: height=%d hash=%s\n", block.Header.Height, block.Hash().String())
        return nil
}