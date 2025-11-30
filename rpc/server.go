// SPDX-License-Identifier: MIT
// Dev: KryperAI

package rpc

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
	"math/big"

	"krypper-chain/node"
	"krypper-chain/types"
)

type Server struct {
	node *node.Node
}

func NewServer(n *node.Node) *Server { return &Server{node:n} }

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/chain/head", s.handleHead)
	mux.HandleFunc("/account/balance", s.handleBalance)
	mux.HandleFunc("/tx/send", s.handleSend)

	log.Println("RPC online:",addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleHead(w http.ResponseWriter,_ *http.Request){
	b:=s.node.Chain.Head()
	o:=map[string]any{
		"height":b.Header.Height,
		"hash":b.Hash().String(),
		"timestamp":time.Unix(b.Header.Timestamp,0).String(),
	}
	json.NewEncoder(w).Encode(o)
}

func (s *Server) handleBalance(w http.ResponseWriter,r *http.Request){
	a:=r.URL.Query().Get("address")
	addr,_:=types.ParseAddress(a)
	b:=s.node.State.GetBalance(addr)
	n:=s.node.State.GetNonce(addr)
	o:=map[string]any{ "address":a, "balance":b.String(), "nonce":n }
	json.NewEncoder(w).Encode(o)
}

func (s *Server) handleSend(w http.ResponseWriter,r *http.Request){
	raw,_ := io.ReadAll(r.Body)

	var in struct{
		ChainId uint64 `json:"chainId"`
		Nonce uint64 `json:"nonce"`
		To string `json:"to"`
		Value string `json:"value"`
		GasPrice string `json:"gasPrice"`
		GasLimit uint64 `json:"gasLimit"`
		Data string `json:"data"`
		R string `json:"r"`
		S string `json:"s"`
		V uint64 `json:"v"`
	}
	_ = json.Unmarshal(raw,&in)

	to,_ := types.ParseAddress(in.To)
	val,_ := new(big.Int).SetString(in.Value,10)
	gp,_ := new(big.Int).SetString(in.GasPrice,10)

	tx := types.NewTransferTx(in.ChainId,in.Nonce,to,val,gp,in.GasLimit,nil)
	tx.Signature = types.SigFromHex(in.R,in.S,in.V)

	err := s.node.Mempool.AddTx(tx)
	out:=map[string]any{};if err!=nil{
		out["status"]="rejected";out["error"]=err.Error()
	}else{
		out["status"]="accepted";out["hash"]=tx.Hash().String()
	}
	json.NewEncoder(w).Encode(out)
}