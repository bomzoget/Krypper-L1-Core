// SPDX-License-Identifier: MIT
// Dev: KryperAI

package node

import (
	"log"
	"sync"
	"time"

	"krypper-chain/types"
)

// Node represents a single KRYPPER L1 execution node with mining loop.
type Node struct {
	mu sync.RWMutex

	Chain        *types.Blockchain
	State        *types.StateDB
	Mempool      *types.Mempool
	Executor     *types.Executor
	MinerAddress types.Address

	// Tier-3 mobile witnesses queued for the next blocks
	WitnessQueue []types.Witness

	Running   bool
	BlockTime time.Duration
}

// NewNode creates a new node instance.
func NewNode(
	chain *types.Blockchain,
	state *types.StateDB,
	mem *types.Mempool,
	exec *types.Executor,
	minerAddr types.Address,
) *Node {
	return &Node{
		Chain:        chain,
		State:        state,
		Mempool:      mem,
		Executor:     exec,
		MinerAddress: minerAddr,
		BlockTime:    3 * time.Second,
		WitnessQueue: make([]types.Witness, 0),
	}
}

// Start begins the mining loop.
func (n *Node) Start() {
	n.mu.Lock()
	if n.Running {
		n.mu.Unlock()
		return
	}
	n.Running = true
	n.mu.Unlock()

	log.Println("NODE: started mining loop")
	go n.minerLoop()
}

// Stop stops the mining loop.
func (n *Node) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Running = false
	log.Println("NODE: stopped")
}

// AddWitness enqueues a tier-3 witness attestation.
func (n *Node) AddWitness(w types.Witness) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.WitnessQueue = append(n.WitnessQueue, w)
}

// minerLoop produces blocks periodically while the node is running.
func (n *Node) minerLoop() {
	ticker := time.NewTicker(n.BlockTime)
	defer ticker.Stop()

	for {
		n.mu.RLock()
		running := n.Running
		n.mu.RUnlock()

		if !running {
			return
		}

		<-ticker.C

		// pick transactions from mempool
		txs := n.Mempool.PopForBlock(100)
		if len(txs) == 0 {
			continue
		}

		n.createBlock(txs)
	}
}

// createBlock builds, dry-runs, and submits a new block to the chain.
func (n *Node) createBlock(txs []*types.Transaction) {
	head := n.Chain.Head()
	if head == nil {
		log.Println("NODE: cannot mine, no genesis head")
		return
	}

	// pick a witness (tier-3 mobile) if any are queued
	var witnessAddr types.Address
	n.mu.Lock()
	if len(n.WitnessQueue) > 0 {
		w := n.WitnessQueue[0]
		witnessAddr = w.Address
		n.WitnessQueue = n.WitnessQueue[1:]
	}
	n.mu.Unlock()

	header := types.BlockHeader{
		ParentHash: head.Hash(),
		Height:     head.Header.Height + 1,
		Timestamp:  time.Now().Unix(),
		Proposer:   n.MinerAddress,
		Witness:    witnessAddr,
		GasLimit:   30_000_000,
	}

	// dry-run execution to compute StateRoot
	snap := n.State.Snapshot()

	n.Executor.SetCoinbase(n.MinerAddress)

	for _, tx := range txs {
		_, err := n.Executor.ExecuteTx(tx)
		if err != nil {
			n.State.RevertToSnapshot(snap)
			log.Printf("NODE: dry-run failed for tx %s: %v\n", tx.Hash().String(), err)
			return
		}
	}

	header.StateRoot = n.State.StateRoot()

	// revert dry-run; Blockchain.AddBlock will execute again and commit for real
	n.State.RevertToSnapshot(snap)

	block := types.NewBlock(&header, txs)
	block.ComputeTxRoot()

	if err := n.Chain.AddBlock(block); err != nil {
		log.Printf("NODE: failed to add block #%d: %v\n", header.Height, err)
		return
	}

	log.Printf("NODE: mined block #%d hash=%s witness=%s\n",
		header.Height, block.Hash().String(), header.Witness.String())
}