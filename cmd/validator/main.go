// SPDX-License-Identifier: MIT
// Dev: KryperAI

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"krypper-chain/types"
)

// Default RPC endpoint
const defaultRPC = "http://localhost:8545"

type chainHeadResponse struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

func main() {
	rpcURL := flag.String("rpc", defaultRPC, "RPC base URL (http://host:port)")
	privHex := flag.String("priv", "", "validator private key (hex)")
	chainID := flag.Uint64("chain-id", 1, "chain ID")
	interval := flag.Duration("interval", 5*time.Second, "poll interval for new blocks")
	flag.Parse()

	if *privHex == "" {
		log.Fatal("missing -priv private key")
	}

	// Load private key
	privKey, addr := mustLoadKey(*privHex)
	fmt.Println("=== KRYPPER TIER-2 VALIDATOR ===")
	fmt.Println("Validator address:", addr.String())
	fmt.Println("RPC endpoint:", *rpcURL)
	fmt.Println("Chain ID:", *chainID)
	fmt.Println("Poll interval:", interval.String())
	fmt.Println()

	lastHeight := uint64(0)

	for {
		head, err := fetchChainHead(*rpcURL)
		if err != nil {
			log.Println("head error:", err)
			time.Sleep(*interval)
			continue
		}

		// No new block
		if head.Height == 0 || head.Height == lastHeight {
			time.Sleep(*interval)
			continue
		}

		// Parse block hash
		blockHash, err := parseHash(head.Hash)
		if err != nil {
			log.Println("invalid head hash:", err)
			time.Sleep(*interval)
			continue
		}

		// Create and sign validator vote
		vote, err := types.SignValidatorVote(privKey, *chainID, head.Height, blockHash)
		if err != nil {
			log.Println("sign vote error:", err)
			time.Sleep(*interval)
			continue
		}

		// Send vote to node
		if err := submitVote(*rpcURL, vote); err != nil {
			log.Println("submit vote error:", err)
		} else {
			log.Printf("✔ vote submitted for height=%d hash=%s\n", vote.Height, vote.BlockHash.String())
			lastHeight = head.Height
		}

		time.Sleep(*interval)
	}
}

func fetchChainHead(rpcURL string) (*chainHeadResponse, error) {
	url := strings.TrimRight(rpcURL, "/") + "/chain/head"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chain head error: %s", string(body))
	}

	var head chainHeadResponse
	if err := json.NewDecoder(resp.Body).Decode(&head); err != nil {
		return nil, err
	}
	return &head, nil
}

func submitVote(rpcURL string, vote *types.ValidatorVote) error {
	data, err := json.Marshal(vote)
	if err != nil {
		return err
	}

	url := strings.TrimRight(rpcURL, "/") + "/validator/attest"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("validator/attest error: %s", string(body))
	}

	return nil
}

func mustLoadKey(hexStr string) (*ecdsa.PrivateKey, types.Address) {
	hexStr = strings.TrimSpace(hexStr)
	if strings.HasPrefix(hexStr, "0x") || strings.HasPrefix(hexStr, "0X") {
		hexStr = hexStr[2:]
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		log.Fatalf("invalid private key hex: %v", err)
	}

	key, err := crypto.ToECDSA(b)
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}
	addr := types.PubKeyToAddress(&key.PublicKey)

	return key, addr
}

func parseHash(s string) (types.Hash, error) {
	var h types.Hash
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if len(s) != 64 {
		return h, fmt.Errorf("invalid hash length")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return h, err
	}
	copy(h[:], b)
	return h, nil
}