// SPDX-License-Identifier: MIT
// Dev: KryperAI

package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"krypper-chain/types"
)

type GenesisAccount struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
}

type GenesisValidator struct {
	Address string `json:"address"`
	Stake   string `json:"stake"`
}

type Genesis struct {
	ChainID    uint64             `json:"chain_id"`
	Alloc      []GenesisAccount   `json:"alloc"`
	Validators []GenesisValidator `json:"validators"`
}

func LoadGenesis(path string) (*Genesis, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open genesis file: %w", err)
	}
	var g Genesis
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, fmt.Errorf("invalid genesis json: %w", err)
	}
	return &g, nil
}

// ApplyGenesis boots initial state from genesis.json
func ApplyGenesis(state *types.StateDB, cfg Config, g *Genesis) ([]types.Address, error) {
	if g.ChainID != 0 && g.ChainID != cfg.Chain.ChainID {
		return nil, fmt.Errorf("genesis.chain_id mismatch: genesis=%d config=%d", g.ChainID, cfg.Chain.ChainID)
	}

	// 1. Alloc – premint supply
	for _, acc := range g.Alloc {
		addr, err := types.ParseAddress(acc.Address)
		if err != nil {
			return nil, fmt.Errorf("bad genesis account %s: %w", acc.Address, err)
		}
		amount, ok := new(big.Int).SetString(acc.Balance, 10)
		if !ok {
			return nil, fmt.Errorf("non-numeric balance for %s", acc.Address)
		}
		if amount.Sign() < 0 {
			return nil, fmt.Errorf("negative genesis balance for %s", acc.Address)
		}
		if err := state.Mint(addr, amount); err != nil {
			return nil, fmt.Errorf("mint failed %s: %w", acc.Address, err)
		}
	}

	// 2. RewardPool – ensure account exists
	if cfg.Chain.RewardPoolAddr != "" {
		rp, err := types.ParseAddress(cfg.Chain.RewardPoolAddr)
		if err != nil {
			return nil, fmt.Errorf("invalid reward_pool address: %w", err)
		}
		if err := ensureAccountExists(state, rp); err != nil {
			return nil, err
		}
	}

	// 3. Validators – this section fixed and finalized
	var validators []types.Address
	for _, v := range g.Validators {
		addr, err := types.ParseAddress(v.Address)
		if err != nil {
			return nil, fmt.Errorf("invalid validator %s", v.Address)
		}
		stake, ok := new(big.Int).SetString(v.Stake, 10)
		if !ok || stake.Sign() <= 0 {
			return nil, fmt.Errorf("invalid stake for %s", v.Address)
		}
		if err := ensureAccountExists(state, addr); err != nil {
			return nil, err
		}
		if err := state.SetStake(addr, stake); err != nil {
			return nil, fmt.Errorf("stake assign failed: %w", err)
		}
		validators = append(validators, addr)
	}

	return validators, nil
}

func ensureAccountExists(state *types.StateDB, addr types.Address) error {
	if state.GetAccount(addr) != nil {
		return nil
	}
	return state.CreateAccount(addr)
}