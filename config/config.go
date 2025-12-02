// SPDX-License-Identifier: MIT
// Dev: KryperAI

package config

import (
        "encoding/hex"
        "fmt"
        "os"
        "strconv"
        "strings"

        "github.com/joho/godotenv"
        "krypper-chain/types"
)

type Config struct {
        MinerAddress types.Address
        MinerPrivKey []byte
        RPCPort      string
        NetworkID    uint64
        GenesisPath  string
        PeerList     string
}

func Load() (*Config, error) {
        _ = godotenv.Load()

        cfg := &Config{
                RPCPort:     getEnv("RPC_PORT", "8000"),
                NetworkID:   getEnvUint64("NETWORK_ID", 1),
                GenesisPath: getEnv("GENESIS_PATH", ""),
                PeerList:    getEnv("PEER_LIST", ""),
        }

        minerAddrStr := cleanEnvValue(os.Getenv("MINER_ADDRESS"))
        minerPrivStr := cleanEnvValue(os.Getenv("MINER_PRIVATE_KEY"))

        if minerPrivStr != "" {
                privBytes, err := hex.DecodeString(minerPrivStr)
                if err != nil {
                        return nil, fmt.Errorf("invalid MINER_PRIVATE_KEY: %w", err)
                }
                cfg.MinerPrivKey = privBytes

                addr, err := types.AddressFromPrivateKey(privBytes)
                if err != nil {
                        return nil, fmt.Errorf("failed to derive address from private key: %w", err)
                }
                cfg.MinerAddress = addr
        } else if minerAddrStr != "" {
                addr, err := types.HexToAddress(minerAddrStr)
                if err != nil {
                        return nil, fmt.Errorf("invalid MINER_ADDRESS: %w", err)
                }
                cfg.MinerAddress = addr
        } else {
                _, addr, _ := types.GenerateKey()
                cfg.MinerAddress = addr
        }

        return cfg, nil
}

func (c *Config) Print() {
        fmt.Println("=== Configuration ===")
        fmt.Printf("  Miner Address: %s\n", c.MinerAddress.String())
        fmt.Printf("  RPC Port:      %s\n", c.RPCPort)
        fmt.Printf("  Network ID:    %d\n", c.NetworkID)
        if c.GenesisPath != "" {
                fmt.Printf("  Genesis Path:  %s\n", c.GenesisPath)
        }
        if c.PeerList != "" {
                fmt.Printf("  Peer List:     %s\n", c.PeerList)
        }
        fmt.Println("=====================")
}

func getEnv(key, defaultVal string) string {
        if val := os.Getenv(key); val != "" {
                return val
        }
        return defaultVal
}

func getEnvUint64(key string, defaultVal uint64) uint64 {
        if val := os.Getenv(key); val != "" {
                if parsed, err := strconv.ParseUint(val, 10, 64); err == nil {
                        return parsed
                }
        }
        return defaultVal
}

func cleanEnvValue(val string) string {
        val = strings.TrimSpace(val)
        if idx := strings.Index(val, "#"); idx != -1 {
                val = strings.TrimSpace(val[:idx])
        }
        return val
}
