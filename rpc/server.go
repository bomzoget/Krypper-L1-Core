// SPDX-License-Identifier: MIT
// Dev: KryperAI

package rpc

import (
	"encoding/json"
	"log"
	"net/http"

	"krypper-chain/node"
	"krypper-chain/types"
)

type Server struct {
	node *node.Node
}

func NewServer(n *node.Node) *Server {
	return &Server{node: n}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// Public RPC
	mux.HandleFunc("/tx/send", s.handleSendTx)
	mux.HandleFunc("/account/balance", s.handleBalance)
	mux.HandleFunc("/chain/head", s.handleHead)

	// Validator / Witness
	mux.HandleFunc("/witness/submit", s.handleSubmitWitness)
	mux.HandleFunc("/validator/vote", s.handleSubmitVote)

	log.Println("RPC Active", addr)
	return http.ListenAndServe(addr, mux)
}

// ============ TX SUBMIT ============
func (s *Server) handleSendTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}

	var tx types.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}

	// REAL METHOD WE HAVE IN SYSTEM ✔
	from, err := types.RecoverTxSender(&tx)
	if err != nil {
		http.Error(w, "invalid signature", 400)
		return
	}

	if err := s.node.Mempool.AddTx(&tx); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	log.Printf("TX ACCEPTED [%s → %s]\n", from.String(), tx.To.String())

	json.NewEncoder(w).Encode(map[string]any{
		"status": "accepted",
		"from":   from.String(),
		"to":     tx.To.String(),
		"hash":   tx.Hash().String(),
	})
}

// ============ ACCOUNT =============
func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	addrHex := r.URL.Query().Get("address")
	addr, _ := types.ParseAddress(addrHex)

	bal := s.node.State.GetBalance(addr)
	nonce := s.node.State.GetNonce(addr)

	json.NewEncoder(w).Encode(map[string]any{
		"address": addr.String(),
		"balance": bal.String(),
		"nonce":   nonce,
	})
}

// ============ HEAD =============
func (s *Server) handleHead(w http.ResponseWriter, r *http.Request) {
	h := s.node.Chain.Head()

	json.NewEncoder(w).Encode(map[string]any{
		"height": h.Header.Height,
		"hash":   h.Hash().String(),
	})
}

// ============ WITNESS =============
func (s *Server) handleSubmitWitness(w http.ResponseWriter, r *http.Request) {
	var wtx types.Witness
	if err := json.NewDecoder(r.Body).Decode(&wtx); err != nil {
		http.Error(w, "invalid witness json", 400)
		return
	}

	s.node.AddWitness(wtx)

	json.NewEncoder(w).Encode(map[string]any{
		"stored":  true,
		"address": wtx.Address.String(),
	})
}

// ============ VALIDATOR =============
func (s *Server) handleSubmitVote(w http.ResponseWriter, r *http.Request) {
	var vote types.ValidatorVote
	if err := json.NewDecoder(r.Body).Decode(&vote); err != nil {
		http.Error(w, "invalid vote json", 400)
		return
	}

	s.node.AddValidatorVote(vote)

	json.NewEncoder(w).Encode(map[string]any{
		"accepted": true,
	})
}