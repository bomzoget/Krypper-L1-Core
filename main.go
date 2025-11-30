package main

import (
	"fmt"
	"log"
	"math/big"
	"time"

	"krypper-chain/types"
)

func main() {
	fmt.Println("Launching KRYPPER Node Simulation...")

	// Core components
	state := types.NewStateDB()
	mempool := types.NewMempool(state)

	minerKey, minerAddr, _ := types.GenerateKey()
	poolAddr := types.Address{0x99}

	config := types.ChainConfig{
		ChainID:        1,
		Coinbase:       minerAddr,
		RewardPoolAddr: poolAddr,
		PoolShare:      10,
	}

	executor := types.NewExecutor(state, config)
	chain := types.NewBlockchain(state, executor)

	fmt.Println("Miner Address:", minerAddr)
	fmt.Println("Fee Reserve Pool:", poolAddr)

	// ---------------- GENESIS ---------------- //
	genesisFund := new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(1e18))
	state.Mint(minerAddr, genesisFund)

	gHeader := types.BlockHeader{
		Height:    0,
		Timestamp: time.Now().Unix(),
		StateRoot: state.StateRoot(),
		Proposer:  minerAddr,
		GasLimit:  30_000_000,
	}

	genesis := types.NewBlock(gHeader, []*types.Transaction{})
	genesis.ComputeTxRoot()

	if err := chain.AddBlock(genesis); err != nil {
		log.Fatal("GENESIS FAILED:", err)
	} else {
		fmt.Println("GENESIS COMMITTED")
	}

	// -------------- TX GENERATION -------------- //
	userKey, userAddr, _ := types.GenerateKey()

	for i := 0; i < 5; i++ {
		value := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18))
		gasPrice := big.NewInt(int64(1_000_000_000 + (i * 200_000)))

		nonce := state.GetNonce(minerAddr)
		tx := types.NewTransferTx(1, nonce+uint64(i), userAddr, value, gasPrice, 21000, nil)

		if err := types.SignTransaction(tx, minerKey); err != nil {
			log.Fatal("SIGN ERROR:", err)
		}
		if err := mempool.AddTx(tx); err != nil {
			log.Println("TX REJECTED →", err)
		}
	}

	fmt.Println("Mempool Size:", mempool.Count())

	// -------------- BLOCK MINING -------------- //
	fmt.Println("Mining Block #1...")

	selected := mempool.PopForBlock(3)
	snap := state.Snapshot()

	for _, tx := range selected {
		executor.SetCoinbase(minerAddr)
		executor.ExecuteTx(tx)
	}

	newRoot := state.StateRoot()
	state.RevertToSnapshot(snap)

	b1 := types.BlockHeader{
		ParentHash: genesis.Hash(),
		Height:     1,
		Timestamp:  time.Now().Unix(),
		StateRoot:  newRoot,
		Proposer:   minerAddr,
		GasLimit:   30_000_000,
	}
	block1 := types.NewBlock(b1, selected)
	block1.ComputeTxRoot()

	if err := chain.AddBlock(block1); err != nil {
		log.Fatal("BLOCK1 REJECTED:", err)
	} else {
		fmt.Println("BLOCK1 ACCEPTED")
	}

	// ----------- FINAL INFO ----------- //
	fmt.Println("Final State:")
	fmt.Println("User Balance:", state.GetBalance(userAddr))
	fmt.Println("Miner Balance:", state.GetBalance(minerAddr))
	fmt.Println("Reward Pool:", state.GetBalance(poolAddr))
	fmt.Println("Chain Height:", chain.Head().Header.Height)
}