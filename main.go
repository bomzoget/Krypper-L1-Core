// SPDX-License-Identifier: MIT
// Dev: KryperAI

package main

import (
	"fmt"
	"log"
	"math/big"
	"time"

	"krypper-chain/node"
	"krypper-chain/rpc"
	"krypper-chain/types"
)

func main() {
	fmt.Println("=== KRYPPER L1 NODE BOOT ===")

	// Core = State + Mempool
	state := types.NewStateDB()
	mempool := types.NewMempool(state)

	// Tier1 Miner Wallet
	minerKey, minerAddr, _ := types.GenerateKey()
	fmt.Println("Miner Address:", minerAddr.String())

	// Reward Pool Address
	var poolAddr types.Address
	poolAddr[0] = 0x99

	// Trinity Reward Distribution
	cfg := types.ChainConfig{
		ChainID:    1,
		RewardPool: poolAddr,

		ShareTier1: 70,
		ShareTier2: 20,
		ShareTier3: 5,
		SharePool:  5,
	}

	// Connect Engine
	exec := types.NewExecutor(state, cfg)
	chain := types.NewBlockchain(state, exec)

	// ========== GENESIS BLOCK ==========
	genesisAmount := new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(1e18))
	_ = state.AddBalance(minerAddr, genesisAmount)

	genHeader := &types.BlockHeader{
		ParentHash: types.ZeroHash(), // FIXED
		Height:     0,
		Timestamp:  time.Now().Unix(),
		StateRoot:  state.StateRoot(),
		TxRoot:     types.ZeroHash(), // FIXED
		GasLimit:   30_000_000,
		Proposer:   minerAddr,
	}

	genesis := types.NewBlock(genHeader, []*types.Transaction{})

	if err := chain.AddBlock(genesis); err != nil {
		log.Fatalf("GENESIS FAILED: %v", err)
	}
	fmt.Println("GENESIS OK — HEIGHT =", chain.Head().Header.Height)

	// Start Node (Auto-Mining)
	n := node.NewNode(chain, state, mempool, exec, minerAddr)
	n.BlockTime = 5 * time.Second
	n.Start()

	// RPC API ENABLED
	srv := rpc.NewServer(n)
	go func() {
		fmt.Println("RPC LISTEN → :8545")
		if err := srv.Start(":8545"); err != nil {
			log.Fatalf("RPC ERROR: %v", err)
		}
	}()

	fmt.Println("NODE ACTIVE — MINING LIVE — RPC READY")
	select {} // infinite run
}