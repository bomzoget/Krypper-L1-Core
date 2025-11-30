// SPDX-License-Identifier: MIT
// Dev: KryperAI

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
)

type ChainConfig struct {
	ChainID       uint64 `json:"chain_id"`
	NetworkID     uint64 `json:"network_id"`
	BlockTime     uint64 `json:"block_time"`
	MaxBlockSize  uint64 `json:"max_block_size"`
	BlockGasLimit uint64 `json:"block_gas_limit"`

	RewardPoolAddr string `json:"reward_pool"`
	BaseReward     string `json:"base_reward"`

	ShareTier1 uint64 `json:"share_t1"`
	ShareTier2 uint64 `json:"share_t2"`
	ShareTier3 uint64 `json:"share_t3"`
	SharePool  uint64 `json:"share_pool"`

	ValidatorCount int    `json:"validator_count"`
	MinStake       string `json:"min_stake"`
}

type NodeConfig struct {
	MinerAddress     string   `json:"miner"`
	P2PListenAddress string   `json:"p2p"`
	RPCListenAddress string   `json:"rpc"`
	Bootnodes        []string `json:"bootnodes"`
	DataDir          string   `json:"data_dir"`
	GenesisFile      string   `json:"genesis"`
	LogLevel         string   `json:"log"`
}

type Config struct {
	Chain ChainConfig `json:"chain"`
	Node  NodeConfig  `json:"node"`
}

func DefaultConfig() Config {
	return Config{
		Chain: ChainConfig{
			ChainID:        1,
			NetworkID:      1,
			BlockTime:      5,
			MaxBlockSize:   2000000,
			BlockGasLimit:  10000000,
			BaseReward:     "5000000000000000000",
			RewardPoolAddr: "0x0000000000000000000000000000000000000099",
			ShareTier1:     60,
			ShareTier2:     25,
			ShareTier3:     5,
			SharePool:      10,
			ValidatorCount: 21,
			MinStake:       "1000000000000000000000",
		},
		Node: NodeConfig{
			MinerAddress:     "",
			P2PListenAddress: "0.0.0.0:30303",
			RPCListenAddress: "0.0.0.0:8545",
			DataDir:          "./chaindata",
			GenesisFile:      "./config/genesis.json",
			LogLevel:         "info",
		},
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("cannot open config file: %w", err)
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return cfg, fmt.Errorf("invalid config json: %w", err)
		}
	}

	if err := applyEnvOverrides(&cfg); err != nil {
		return cfg, fmt.Errorf("environment override error: %w", err)
	}

	return cfg, validate(cfg)
}

func applyEnvOverrides(cfg *Config) error {
	parseUint := func(key string, field *uint64) error {
		v := os.Getenv(key)
		if v == "" {
			return nil
		}
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return fmt.Errorf("%s invalid numeric value: %s", key, v)
		}
		*field = n
		return nil
	}

	if err := parseUint("KRYPPER_CHAIN_ID", &cfg.Chain.ChainID); err != nil { return err }
	if err := parseUint("KRYPPER_BLOCK_TIME", &cfg.Chain.BlockTime); err != nil { return err }
	if err := parseUint("KRYPPER_GAS_LIMIT", &cfg.Chain.BlockGasLimit); err != nil { return err }
	if err := parseUint("KRYPPER_SHARE_T1", &cfg.Chain.ShareTier1); err != nil { return err }
	if err := parseUint("KRYPPER_SHARE_T2", &cfg.Chain.ShareTier2); err != nil { return err }
	if err := parseUint("KRYPPER_SHARE_T3", &cfg.Chain.ShareTier3); err != nil { return err }
	if err := parseUint("KRYPPER_SHARE_POOL", &cfg.Chain.SharePool); err != nil { return err }

	if v := os.Getenv("KRYPPER_REWARD_POOL"); v != "" { cfg.Chain.RewardPoolAddr = v }
	if v := os.Getenv("KRYPPER_MINER"); v != "" { cfg.Node.MinerAddress = v }
	if v := os.Getenv("KRYPPER_RPC"); v != "" { cfg.Node.RPCListenAddress = v }
	if v := os.Getenv("KRYPPER_P2P"); v != "" { cfg.Node.P2PListenAddress = v }
	if v := os.Getenv("KRYPPER_DATA_DIR"); v != "" { cfg.Node.DataDir = v }

	if v := os.Getenv("KRYPPER_BOOTNODES"); v != "" {
		parts := strings.Split(v, ",")
		var nodes []string
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				nodes = append(nodes, trimmed)
			}
		}
		if len(nodes) > 0 {
			cfg.Node.Bootnodes = nodes
		}
	}

	return nil
}

func validate(c Config) error {
	if c.Chain.ChainID == 0 {
		return errors.New("chain_id must be > 0")
	}

	sum := c.Chain.ShareTier1 + c.Chain.ShareTier2 + c.Chain.ShareTier3 + c.Chain.SharePool
	if sum > 100 {
		return fmt.Errorf("fee distribution greater than 100 percent")
	}

	if _, ok := new(big.Int).SetString(c.Chain.BaseReward, 10); !ok {
		return fmt.Errorf("invalid base_reward format: %s", c.Chain.BaseReward)
	}
	if _, ok := new(big.Int).SetString(c.Chain.MinStake, 10); !ok {
		return fmt.Errorf("invalid min_stake format: %s", c.Chain.MinStake)
	}

	if !strings.HasPrefix(c.Chain.RewardPoolAddr, "0x") {
		return errors.New("reward_pool address invalid format")
	}

	if c.Node.MinerAddress != "" && !strings.HasPrefix(c.Node.MinerAddress, "0x") {
		return errors.New("miner address invalid format")
	}

	if c.Node.DataDir == "" {
		return errors.New("data_dir cannot be empty")
	}

	return nil
}