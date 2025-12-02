// SPDX-License-Identifier: MIT
// Dev: KryperAI

package main

import (
        "flag"
        "fmt"
        "log"
        "math/big"
        "strings"

        "krypper-chain/node"
        "krypper-chain/p2p"
        "krypper-chain/rpc"
        "krypper-chain/types"
)

func main() {
        // Command args
        rpcPort := flag.String("port", "8000", "RPC port")
        peerList := flag.String("peers", "", "Comma separated peer URLs")
        flag.Parse()

        fmt.Println("=== KRYPPER NODE START ===")

        // Core system
        state := types.NewStateDB()
        mempool := types.NewMempool(state)

        // Miner identity (unique per node)
        _, minerAddr, _ := types.GenerateKey()
        fmt.Println("Miner:", minerAddr.String())

        // Trinity economy config
        var rewardPool types.Address
        rewardPool[0] = 0xAA

        cfg := types.ChainConfig{
                ChainID:    1,
                RewardPool: rewardPool,
                ShareTier1: 70,
                ShareTier2: 20,
                ShareTier3: 5,
                SharePool:  5,
        }

        exec := types.NewExecutor(state, cfg)
        chain := types.NewBlockchain(state, exec)

        // ------------------------------
        // DETERMINISTIC GENESIS
        // ------------------------------

        var gAddress types.Address
        gAddress[0] = 0x11 // fixed genesis holder

        amount := new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(1e18))
        state.Mint(gAddress, amount)

        genHeader := &types.BlockHeader{
                ParentHash: types.ZeroHash(),
                Height:     0,
                Timestamp:  1700000000, // fixed time + reproducible
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

        // ------------------------------
        // P2P
        // ------------------------------
        peers := []string{}
        if *peerList != "" {
                peers = strings.Split(*peerList, ",")
        }
        _ = p2p.NewManager(peers)

        // ------------------------------
        // NODE
        // ------------------------------
        n := node.NewNode(chain, state, mempool, exec, minerAddr)
        n.Start()

        // ------------------------------
        // RPC
        // ------------------------------
        server := rpc.NewServer(n)
        go func() {
                addr := ":" + *rpcPort
                fmt.Println("RPC:", addr)
                if err := server.Start(addr); err != nil {
                        log.Fatal(err)
                }
        }()

        fmt.Println("NODE RUNNING")
        select {}
}