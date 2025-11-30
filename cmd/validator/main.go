// SPDX-License-Identifier: MIT
// Dev: KryperAI

package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"krypper-chain/types"
)

type headResponse struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

func main() {
	nodeRPC := flag.String("node", "http://localhost:8545", "Node RPC URL")
	privHex := flag.String("priv", "", "Validator private key (hex)")
	chainID := flag.Uint64("chain-id", 1, "Chain ID")
	flag.Parse()

	if *privHex == "" {
		log.Fatal("missing -priv")
	}

	privKey, addr := loadKey(*privHex)
	fmt.Println("Validator running")
	fmt.Println("Address:", addr.String())
	fmt.Println("Node RPC:", *nodeRPC)

	var lastHeight uint64

	for {
		h, err := fetchHead(*nodeRPC)
		if err != nil {
			log.Println("fetch head error:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if h.Height == 0 || h.Height <= lastHeight {
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Printf("Validating block #%d (%s)\n", h.Height, h.Hash)

		blockHash, err := parseHash(h.Hash)
		if err != nil {
			log.Println("invalid head hash:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		vote, err := types.SignValidatorVote(privKey, *chainID, h.Height, blockHash)
		if err != nil {
			log.Println("sign vote error:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if err := submitVote(*nodeRPC, vote); err != nil {
			log.Println("submit vote error:", err)
		} else {
			fmt.Println("Vote submitted")
			lastHeight = h.Height
		}

		time.Sleep(2 * time.Second)
	}
}

func fetchHead(rpcURL string) (headResponse, error) {
	var out headResponse

	resp, err := http.Get(strings.TrimRight(rpcURL, "/") + "/chain/head")
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return out, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

func submitVote(rpcURL string, vote *types.ValidatorVote) error {
	data, err := json.Marshal(vote)
	if err != nil {
		return err
	}

	url := strings.TrimRight(rpcURL, "/") + "/validator/vote"
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
		return fmt.Errorf("rpc error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func loadKey(hexStr string) (*ecdsa.PrivateKey, types.Address) {
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