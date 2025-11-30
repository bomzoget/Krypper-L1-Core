// SPDX-License-Identifier: MIT
// Dev: KrypperAI

package rpc

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"krypper-chain/node"
	"krypper-chain/types"
)

type Server struct{ n *node.Node }

func NewServer(n *node.Node)*Server{ return &Server{n:n} }

func (s *Server) Start(addr string) error {
	http.HandleFunc("/tx/send", s.handleSendTx)
	http.HandleFunc("/account/balance", s.handleBalance)
	http.HandleFunc("/chain/head", s.handleHead)
	return http.ListenAndServe(addr,nil)
}

func (s *Server) handleHead(w http.ResponseWriter,_ *http.Request){
	json.NewEncoder(w).Encode(s.n.Chain.Head())
}

func (s *Server) handleBalance(w http.ResponseWriter,r *http.Request){
	addrStr := r.URL.Query().Get("address")
	addr,_  := types.ParseAddress(addrStr)
	bal := s.n.State.GetBalance(addr)
	nonce:= s.n.State.GetNonce(addr)

	json.NewEncoder(w).Encode(map[string]any{
		"address": addr.String(),
		"balance": bal.String(),
		"nonce":   nonce,
	})
}

func (s *Server) handleSendTx(w http.ResponseWriter,r *http.Request){
	body,_ := io.ReadAll(r.Body)
	var in struct{
		ChainId string `json:"chainId"`
		Nonce uint64  `json:"nonce"`
		To string     `json:"to"`
		Value string  `json:"value"`
		GasPrice string `json:"gasPrice"`
		GasLimit uint64 `json:"gasLimit"`
		Data string `json:"data"`
		R string `json:"r"`
		S string `json:"s"`
		V uint8  `json:"v"`
	}
	json.Unmarshal(body,&in)

	tx := types.DecodeSignedTx(in)
	err := s.n.Mempool.AddTx(tx)

	if err!=nil {
		http.Error(w,err.Error(),400)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"status":"OK","hash":tx.Hash().String()})
}