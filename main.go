// SPDX-License-Identifier: MIT
// Dev: KryperAI

package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"strings"

	"krypper-chain/config"
	"krypper-chain/node"
	"krypper-chain/p2p"
	"krypper-chain/rpc"
	"krypper-chain/types"
)

func main() {
	rpcPortFlag := flag.String("port", "", "RPC port (overrides RPC_PORT env)")
	peerListFlag := flag.String("peers", "", "Comma separated peer URLs (overrides PEER_LIST env)")
	flag.Parse()

	fmt.Println("=== KRYPPER NODE START ===")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("CONFIG ERROR:", err)
	}

	if *rpcPortFlag != "" {
		cfg.RPCPort = *rpcPortFlag
	}
	if *peerListFlag != "" {
		cfg.PeerList = *peerListFlag
	}

	cfg.Print()

	state := types.NewStateDB()
	mempool := types.NewMempool(state)

	minerAddr := cfg.MinerAddress
	fmt.Println("Miner:", minerAddr.String())

	var rewardPool types.Address
	rewardPool[0] = 0xAA

	chainCfg := types.ChainConfig{
		ChainID:    cfg.NetworkID,
		RewardPool: rewardPool,
		ShareTier1: 70,
		ShareTier2: 20,
		ShareTier3: 5,
		SharePool:  5,
	}

	exec := types.NewExecutor(state, chainCfg)
	chain := types.NewBlockchain(state, exec)

	var gAddress types.Address
	gAddress[0] = 0x11

	amount := new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(1e18))
	state.Mint(gAddress, amount)

	genHeader := &types.BlockHeader{
		ParentHash: types.ZeroHash(),
		Height:     0,
		Timestamp:  1700000000,
		StateRoot:  state.StateRoot(),
		TxRoot:     types.ZeroHash(),
		GasLimit:   30_000_000,
		Proposer:   gAddress,
	}

	genesis := types.NewBlock(genHeader, []*types.Transaction{})
	if err := chain.AddBlock(genesis); err != nil {
		log.Fatal("GENESIS:", err)
	}

	fmt.Println("GENESIS OK:", genesis.Hash())

	peers := []string{}
	if cfg.PeerList != "" {
		peers = strings.Split(cfg.PeerList, ",")
	}
	_ = p2p.NewManager(peers)

	n := node.NewNode(chain, state, mempool, exec, minerAddr)
	n.Start()

	server := rpc.NewServer(n)
	go func() {
		addr := ":" + cfg.RPCPort
		fmt.Println("RPC:", addr)
		if err := server.Start(addr); err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Println("NODE RUNNING")
	select {}
}
