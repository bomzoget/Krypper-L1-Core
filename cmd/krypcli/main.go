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
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"krypper-chain/types"
)

const defaultRPC = "http://localhost:8545"

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "new":
		cmdNew()
	case "balance":
		cmdBalance()
	case "send":
		cmdSend()
	default:
		usage()
	}
}

func usage() {
	fmt.Println("krypcli commands:")
	fmt.Println("  krypcli new")
	fmt.Println("  krypcli balance -addr 0x.. [-rpc URL]")
	fmt.Println("  krypcli send -priv HEX -to 0x.. -amount WEI [-gas-price] [-gas-limit] [-rpc URL]")
}

func cmdNew() {
	key, addr, _ := types.GenerateKey()
	priv := hex.EncodeToString(crypto.FromECDSA(key))
	fmt.Println("PrivateKey:", priv)
	fmt.Println("Address:", addr.String())
}

func cmdBalance() {
	fs := flag.NewFlagSet("balance", flag.ExitOnError)
	rpcURL := fs.String("rpc", defaultRPC, "")
	addrStr := fs.String("addr", "", "")
	_ = fs.Parse(os.Args[2:])
	if *addrStr == "" {
		log.Fatal("missing -addr")
	}
	url := fmt.Sprintf("%s/account/balance?address=%s", *rpcURL, *addrStr)
	resp := httpGet(url)
	fmt.Println(string(resp))
}

func cmdSend() {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	rpcURL := fs.String("rpc", defaultRPC, "")
	privHex := fs.String("priv", "", "")
	toStr := fs.String("to", "", "")
	amountStr := fs.String("amount", "", "")
	gasPriceStr := fs.String("gas-price", "1000000000", "")
	gasLimit := fs.Uint64("gas-limit", 21000, "")
	chainID := fs.Uint64("chain-id", 1, "")
	_ = fs.Parse(os.Args[2:])

	if *privHex == "" || *toStr == "" || *amountStr == "" {
		log.Fatal("missing args")
	}

	priv, from, _ := loadKey(*privHex)
	to, _ := parseAddress(*toStr)
	amount, _ := new(big.Int).SetString(*amountStr, 10)
	gasPrice, _ := new(big.Int).SetString(*gasPriceStr, 10)

	nonce := queryNonce(*rpcURL, from)

	tx := types.NewTransferTx(*chainID, nonce, to, amount, gasPrice, *gasLimit, nil)
	_ = types.SignTransaction(tx, priv)

	body := map[string]any{
		"chainId":  tx.ChainID,
		"nonce":    tx.Nonce,
		"to":       tx.To.String(),
		"value":    tx.Value.String(),
		"gasPrice": tx.GasPrice.String(),
		"gasLimit": tx.GasLimit,
		"data":     "0x" + hex.EncodeToString(tx.Data),
		"r":        "0x" + tx.Signature.R.Text(16),
		"s":        "0x" + tx.Signature.S.Text(16),
		"v":        tx.Signature.V,
	}

	j, _ := json.Marshal(body)
	resp := httpPost(*rpcURL+"/tx/send", j)
	fmt.Println(string(resp))
}

func httpGet(url string) []byte {
	r, _ := http.Get(url)
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return b
}

func httpPost(url string, data []byte) []byte {
	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type",application/json")
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return out
}

func loadKey(hexKey string) (*ecdsa.PrivateKey, types.Address, error) {
	h := strings.TrimPrefix(hexKey,"0x")
	b,_ := hex.DecodeString(h)
	k,_ := crypto.ToECDSA(b)
	return k, types.PubKeyToAddress(&k.PublicKey), nil
}

func parseAddress(s string) (types.Address,error) {
	var a types.Address
	s=strings.TrimPrefix(s,"0x")
	b,_ := hex.DecodeString(s)
	copy(a[:],b)
	return a,nil
}

func queryNonce(url string, addr types.Address) uint64 {
	r := httpGet(url+"/account/balance?address="+addr.String())
	var out struct { Nonce uint64 `json:"nonce"` }
	_ = json.Unmarshal(r,&out)
	return out.Nonce
}
