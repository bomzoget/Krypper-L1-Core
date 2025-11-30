// SPDX-License-Identifier: MIT
// Dev: KryperAI

package node

import (
	"log"
	"sync"
	"time"

	"krypper-chain/p2p"
	"krypper-chain/types"
)

type Node struct {
	mu sync.RWMutex

	Chain   *types.Blockchain
	State   *types.StateDB
	Mempool *types.Mempool
	Exec    *types.Executor

	MinerAddress types.Address
	BlockTime    time.Duration

	P2P *p2p.Manager

	running bool
}

func NewNode(
	chain *types.Blockchain,
	state *types.StateDB,
	mem *types.Mempool,
	exec *types.Executor,
	minerAddr types.Address,
	p2pMgr *p2p.Manager,
) *Node {
	return &Node{
		Chain:        chain,
		State:        state,
		Mempool:      mem,
		Exec:         exec,
		MinerAddress: minerAddr,
		BlockTime:    5 * time.Second,
		P2P:          p2pMgr,
	}
}

func (n *Node) Start() {
	n.mu.Lock()
	n.running = true
	n.mu.Unlock()

	log.Println("node: started mining loop")
	go n.minerLoop()
}

func (n *Node) Stop() {
	n.mu.Lock()
	n.running = false
	n.mu.Unlock()
	log.Println("node: stopped")
}

func (n *Node) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.running
}

// BroadcastTx lets RPC layer or others relay a tx to peers.
func (n *Node) BroadcastTx(tx *types.Transaction) {
	if n.P2P == nil || tx == nil {
		return
	}
	n.P2P.BroadcastTx(tx)
}

// BroadcastBlock relays newly accepted block to peers.
func (n *Node) BroadcastBlock(b *types.Block) {
	if n.P2P == nil || b == nil {
		return
	}
	n.P2P.BroadcastBlock(b)
}

func (n *Node) minerLoop() {
	ticker := time.NewTicker(n.BlockTime)
	defer ticker.Stop()

	for {
		if !n.IsRunning() {
			return
		}
		<-ticker.C

		txs := n.Mempool.PopForBlock(100)
		if len(txs) == 0 {
			continue
		}

		log.Printf("node: mining candidate block with %d txs\n", len(txs))
		n.produceBlock(txs)
	}
}

func (n *Node) produceBlock(txs []*types.Transaction) {
	head := n.Chain.Head()
	if head == nil {
		log.Println("node: cannot mine, no genesis")
		return
	}

	header := &types.BlockHeader{
		ParentHash: head.Hash(),
		Height:     head.Header.Height + 1,
		Timestamp:  time.Now().Unix(),
		GasLimit:   30_000_000,
		Proposer:   n.MinerAddress,
		// Validator/Witness can be filled by higher-level logic later
	}

	// dry-run to compute state root
	snap := n.State.Snapshot()
	n.Exec.SetBlock(header)

	for _, tx := range txs {
		if _, err := n.Exec.ExecuteTx(tx); err != nil {
			log.Printf("node: tx execution failed in dry run: %v\n", err)
			n.State.RevertToSnapshot(snap)
			return
		}
	}
	header.StateRoot = n.State.StateRoot()
	n.State.RevertToSnapshot(snap)

	block := types.NewBlock(header, txs)
	block.ComputeTxRoot()

	if err := n.Chain.AddBlock(block); err != nil {
		log.Printf("node: block rejected: %v\n", err)
		return
	}

	log.Printf("node: block #%d mined, hash=%s\n", block.Header.Height, block.Hash().String())

	// relay block to peers
	n.BroadcastBlock(block)
}