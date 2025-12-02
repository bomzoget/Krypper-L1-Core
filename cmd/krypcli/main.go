// SPDX-License-Identifier: MIT
// Dev: KrypperAI

package main

import (
        "bytes"
        "crypto/ecdsa"
        "encoding/hex"
        "encoding/json"
        "flag"
        "fmt"
        "io"
        "math/big"
        "net/http"
        "os"
        "strings"

        "github.com/ethereum/go-ethereum/crypto"
        "krypper-chain/types"
)

const RPC = "http://localhost:8000"

func main() {
        if len(os.Args) < 2 { usage(); return }

        switch os.Args[1] {
        case "new":     newWallet()
        case "balance": balance()
        case "send":    send()
        default: usage()
        }
}

func usage() {
        fmt.Println("krypcli usage:")
        fmt.Println("  krypcli new")
        fmt.Println("  krypcli balance -addr 0x...")
        fmt.Println("  krypcli send -priv HEX -to ADDRESS -amount WEI")
}

// ---------------- KEY GEN ----------------

func newWallet() {
        key, addr, _ := types.GenerateKey()
        fmt.Println("Private:", hex.EncodeToString(key.D.Bytes()))
        fmt.Println("Address:", addr.String())
}

// ---------------- BALANCE ----------------

func balance() {
        fs := flag.NewFlagSet("balance", flag.ExitOnError)
        addrStr := fs.String("addr","", "0x.. address")
        rpcURL := fs.String("rpc",RPC,"node rpc")
        fs.Parse(os.Args[2:])

        addr,_ := parseAddr(*addrStr)
        url := *rpcURL+"/account/balance?address="+addr.String()
        body := httpGet(url)

        fmt.Println(string(body))
}

// ---------------- SEND TX ----------------

func send() {
        fs := flag.NewFlagSet("send",flag.ExitOnError)
        rpcURL := fs.String("rpc",RPC,"node rpc")
        priv := fs.String("priv","", "private hex")
        to   := fs.String("to","",   "receiver")
        amt  := fs.String("amount","", "wei")

        fs.Parse(os.Args[2:])

        key,from,_ := loadKey(*priv)
        toAddr,_   := parseAddr(*to)
        value,_    := new(big.Int).SetString(*amt,10)
        nonce      := getNonce(*rpcURL,from)

        tx := types.NewTransferTx(1,nonce,toAddr,value,big.NewInt(1_000_000_000),21000,nil)
        types.SignTransaction(tx,key)

        req := map[string]any{
                "chainId":"1",
                "nonce": tx.Nonce,
                "to":    tx.To.String(),
                "value": tx.Value.String(),
                "gasPrice": tx.GasPrice.String(),
                "gasLimit": tx.GasLimit,
                "data": "0x"+hex.EncodeToString(tx.Data),
                "r": "0x"+tx.Signature.R.Text(16),
                "s": "0x"+tx.Signature.S.Text(16),
                "v": tx.Signature.V,
        }

        b,_ := json.Marshal(req)
        resp,_ := http.Post(*rpcURL+"/tx/send","application/json",bytes.NewReader(b))
        out,_  := io.ReadAll(resp.Body)

        fmt.Println("TX →",string(out))
}

// ---------------- HELPERS ----------------

func httpGet(url string) []byte { r,_:=http.Get(url); b,_:=io.ReadAll(r.Body); return b }

func loadKey(h string)(*ecdsa.PrivateKey,types.Address,error){
        h=strings.TrimPrefix(h,"0x")
        b,_:=hex.DecodeString(h)
        k,_:=crypto.ToECDSA(b)
        return k,types.PubKeyToAddress(&k.PublicKey),nil
}

func parseAddr(s string)(types.Address,error){
        var a types.Address
        s=strings.TrimPrefix(s,"0x")
        b,_:=hex.DecodeString(s)
        copy(a[:],b)
        return a,nil
}

func getNonce(url string,addr types.Address)uint64{
        b:=httpGet(url+"/account/balance?address="+addr.String())
        var out struct{Nonce uint64 `json:"nonce"` }
        json.Unmarshal(b,&out)
        return out.Nonce
}