// SPDX-License-Identifier: MIT
// Dev: KryperAI

package rpc

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"krypper-chain/node"
	"krypper-chain/types"
)

// Server wraps HTTP handlers around a running node.
type Server struct {
	node *node.Node
}

// NewServer creates a new RPC server.
func NewServer(n *node.Node) *Server {
	return &Server{
		node: n,
	}
}

// Start begins listening on the given address.
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/tx/send", s.handleSendTx)
	mux.HandleFunc("/account/balance", s.handleAccountBalance)
	mux.HandleFunc("/chain/head", s.handleChainHead)

	// tier-3 witness endpoint
	mux.HandleFunc("/witness/submit", s.handleSubmitWitness)

	return http.ListenAndServe(addr, mux)
}

// -------------------------
// /tx/send
// -------------------------

func (s *Server) handleSendTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var tx types.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if err := tx.ValidateBasic(); err != nil {
		http.Error(w, "invalid tx: "+err.Error(), http.StatusBadRequest)
		return
	}

	// optional: verify signature and recover sender
	if _, err := types.RecoverTxSender(&tx); err != nil {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	if err := s.node.Mempool.AddTx(&tx); err != nil {
		http.Error(w, "mempool reject: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp := map[string]any{
		"hash": tx.Hash().String(),
	}
	writeJSON(w, resp)
}

// -------------------------
// /account/balance
// -------------------------

func (s *Server) handleAccountBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	addrStr := r.URL.Query().Get("address")
	if addrStr == "" {
		http.Error(w, "missing address", http.StatusBadRequest)
		return
	}

	addr, err := parseAddress(addrStr)
	if err != nil {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}

	bal := s.node.State.GetBalance(addr)
	nonce := s.node.State.GetNonce(addr)

	resp := map[string]any{
		"address": addr.String(),
		"balance": bal.String(),
		"nonce":   nonce,
	}
	writeJSON(w, resp)
}

// -------------------------
// /chain/head
// -------------------------

func (s *Server) handleChainHead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	head := s.node.Chain.Head()
	if head == nil {
		http.Error(w, "no head block", http.StatusNotFound)
		return
	}

	hdr := head.Header
	resp := map[string]any{
		"height":      hdr.Height,
		"hash":        head.Hash().String(),
		"parentHash":  hdr.ParentHash.String(),
		"stateRoot":   hdr.StateRoot.String(),
		"txRoot":      hdr.TxRoot.String(),
		"timestamp":   hdr.Timestamp,
		"proposer":    hdr.Proposer.String(),
		"witness":     hdr.Witness.String(),
		"gasLimit":    hdr.GasLimit,
		"gasUsed":     hdr.GasUsed,
	}
	writeJSON(w, resp)
}

// -------------------------
// /witness/submit
// -------------------------

func (s *Server) handleSubmitWitness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var wt types.Witness
	if err := json.NewDecoder(r.Body).Decode(&wt); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// TODO: signature validation for witness:
	// - verify that wt.Signature is a valid signature over wt.Hash by wt.Address
	// This requires a helper in types/crypto.go (e.g. VerifyWitness).

	s.node.AddWitness(wt)

	log.Printf("RPC: witness stored addr=%s height=%d\n", wt.Address.String(), wt.BlockHeight)

	resp := map[string]any{
		"stored":  true,
		"height":  wt.BlockHeight,
		"address": wt.Address.String(),
	}
	writeJSON(w, resp)
}

// -------------------------
// helpers
// -------------------------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("RPC: write json error:", err)
	}
}

func parseAddress(s string) (types.Address, error) {
	var addr types.Address

	s = strings.TrimSpace(s)
	if s == "" {
		return addr, nil
	}

	s = strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(s)
	if err != nil {
		return addr, err
	}
	if len(b) != len(addr) {
		return addr, nil
	}
	copy(addr[:], b)
	return addr, nil
}