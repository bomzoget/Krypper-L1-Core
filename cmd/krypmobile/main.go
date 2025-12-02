// SPDX-License-Identifier: MIT
// Dev: KryperAI

package main

import (
        "crypto/ecdsa"
        "encoding/hex"
        "encoding/json"
        "flag"
        "fmt"
        "io"
        "log"
        "math/big"
        "net/http"
        "strings"
        "time"

        "github.com/ethereum/go-ethereum/crypto"
        "krypper-chain/types"
)

const defaultRPC = "http://localhost:8000"

type headResponse struct {
        Header struct {
                Height    uint64 `json:"height"`
                Timestamp int64  `json:"timestamp"`
        } `json:"header"`
}

func main() {
        // flags
        rpcURL := flag.String("rpc", defaultRPC, "RPC URL of KRYPPER node")
        privHex := flag.String("priv", "", "private key (hex) used as mobile miner identity")
        interval := flag.Int("interval", 5, "poll interval in seconds")
        flag.Parse()

        if *privHex == "" {
                log.Fatal("missing -priv (private key hex)")
        }

        // load key
        privKey, addr, err := loadPrivateKey(*privHex)
        if err != nil {
                log.Fatalf("invalid private key: %v", err)
        }

        fmt.Println("=== KRYPPER Tier3 Mobile Miner ===")
        fmt.Println("RPC:", *rpcURL)
        fmt.Println("Address:", addr.String())
        fmt.Println("Interval:", *interval, "sec")

        var lastHeight uint64 = 0

        for {
                h, err := fetchHead(*rpcURL)
                if err != nil {
                        log.Printf("error fetching head: %v", err)
                        time.Sleep(time.Duration(*interval) * time.Second)
                        continue
                }

                if h.Header.Height > lastHeight {
                        fmt.Printf("\n[HEAD] new block height=%d ts=%d\n", h.Header.Height, h.Header.Timestamp)

                        // Create a pseudo header hash (height + timestamp) to sign as witness
                        hash := headerHashMock(h.Header.Height, h.Header.Timestamp)

                        sig, err := crypto.Sign(hash[:], privKey)
                        if err != nil {
                                log.Printf("failed to sign witness: %v", err)
                        } else {
                                fmt.Printf("Witness signature for height %d:\n", h.Header.Height)
                                fmt.Printf("  hash: %s\n", hex.EncodeToString(hash[:]))
                                fmt.Printf("  sig : %s\n", hex.EncodeToString(sig))
                        }

                        lastHeight = h.Header.Height
                }

                time.Sleep(time.Duration(*interval) * time.Second)
        }
}

// headerHashMock builds a 32-byte hash from height + timestamp.
// later you can replace this with real block header hash pulled via RPC.
func headerHashMock(height uint64, ts int64) types.Hash {
        var h types.Hash
        // use big.Int to pack height and ts deterministically
        b := new(big.Int)
        b.Lsh(b.SetUint64(height), 32)
        b.Add(b, big.NewInt(ts))
        buf := b.Bytes()
        copy(h[32-len(buf):], buf)
        return h
}

func fetchHead(rpcURL string) (*headResponse, error) {
        resp, err := http.Get(strings.TrimRight(rpcURL, "/") + "/chain/head")
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
                data, _ := io.ReadAll(resp.Body)
                return nil, fmt.Errorf("rpc status %d: %s", resp.StatusCode, string(data))
        }

        var out headResponse
        if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
                return nil, err
        }
        return &out, nil
}

func loadPrivateKey(hexStr string) (*ecdsa.PrivateKey, types.Address, error) {
        hexStr = strings.TrimSpace(hexStr)
        if strings.HasPrefix(hexStr, "0x") || strings.HasPrefix(hexStr, "0X") {
                hexStr = hexStr[2:]
        }
        b, err := hex.DecodeString(hexStr)
        if err != nil {
                return nil, types.Address{}, err
        }
        key, err := crypto.ToECDSA(b)
        if err != nil {
                return nil, types.Address{}, err
        }
        addr := types.PubKeyToAddress(&key.PublicKey)
        return key, addr, nil
}