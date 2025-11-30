// SPDX-License-Identifier: MIT
// Dev KryperAI

package node

import (
	"log"
	"sync"
	"time"

	"krypper-chain/types"
)

type Node struct {
	mu           sync.RWMutex
	Chain        *types.Blockchain
	State        *types.StateDB
	Mempool      *types.Mempool
	Executor     *types.Executor
	MinerAddress types.Address

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
		Chain:        chain,
		State:        state,
		Mempool:      mem,
		Executor:     exec,
		MinerAddress: minerAddr,
		BlockTime:    3 * time.Second,
	}
}

func (n *Node) Start() {
	n.mu.Lock()
	n.Running = true
	n.mu.Unlock()

	log.Println("NODE: STARTED")
	go n.minerLoop()
}

func (n *Node) Stop() {
	n.mu.Lock()
	n.Running = false
	n.mu.Unlock()
	log.Println("NODE: STOPPED")
}

func (n *Node) minerLoop() {
	ticker := time.NewTicker(n.BlockTime)
	defer ticker.Stop()

	for {
		if !n.Running {
			return
		}

		<-ticker.C

		txs := n.Mempool.PopForBlock(200)

		if len(txs) > 0 {
			n.produceBlock(txs)
		}
	}
}

func (n *Node) produceBlock(txs []*types.Transaction) {
	head := n.Chain.Head()
	if head == nil {
		log.Println("NO GENESIS — BLOCKING MINING")
		return
	}

	header := types.BlockHeader{
		ParentHash: head.Hash(),
		Height:     head.Header.Height + 1,
		Timestamp:  time.Now().Unix(),
		Proposer:   n.MinerAddress,
		GasLimit:   30_000_000,
	}

	snap := n.State.Snapshot()

	n.Executor.SetCoinbase(n.MinerAddress)
	for _, tx := range txs {
		n.Executor.ExecuteTx(tx)
	}

	header.StateRoot = n.State.StateRoot()
	n.State.RevertToSnapshot(snap)

	block := types.NewBlock(header, txs)
	block.ComputeTxRoot()

	if err := n.Chain.AddBlock(block); err != nil {
		log.Printf("BLOCK REJECTED — %v\n", err)
		return
	}

	log.Printf("BLOCK MINED — HEIGHT %d — %s\n", block.Header.Height, block.Hash())
}