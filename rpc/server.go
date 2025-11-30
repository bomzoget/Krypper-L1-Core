// SPDX-License-Identifier: MIT
// Dev: KryperAI

package rpc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"

	"krypper-chain/node"
	"krypper-chain/types"
)

type Server struct {
	node *node.Node
	mux  *http.ServeMux
}

func NewServer(n *node.Node) *Server {
	s := &Server{
		node: n,
		mux:  http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/chain/head", s.handleChainHead)
	s.mux.HandleFunc("/account/balance", s.handleAccountBalance)
	s.mux.HandleFunc("/tx/send", s.handleSendTx)

	// P2P ingress endpoints (used by other nodes)
	s.mux.HandleFunc("/p2p/tx", s.handleP2PTx)
	s.mux.HandleFunc("/p2p/block", s.handleP2PBlock)
}

// -------------------- basic handlers --------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) handleChainHead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	head := s.node.Chain.Head()
	if head == nil {
		httpError(w, http.StatusNotFound, "no blocks yet")
		return
	}

	hdr := head.Header
	writeJSON(w, http.StatusOK, map[string]any{
		"height":    hdr.Height,
		"hash":      head.Hash().String(),
		"stateRoot": hdr.StateRoot.String(),
		"txCount":   len(head.Transactions),
		"proposer":  hdr.Proposer.String(),
	})
}

func (s *Server) handleAccountBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	addrStr := r.URL.Query().Get("address")
	if addrStr == "" {
		httpError(w, http.StatusBadRequest, "missing address")
		return
	}

	addr, err := parseAddress(addrStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid address")
		return
	}

	bal := s.node.State.GetBalance(addr)
	nonce := s.node.State.GetNonce(addr)

	writeJSON(w, http.StatusOK, map[string]any{
		"address": addr.String(),
		"balance": bal.String(),
		"nonce":   nonce,
	})
}

// -------------------- TX: external client --------------------

func (s *Server) handleSendTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "cannot read body")
		return
	}
	defer r.Body.Close()

	var tx types.Transaction
	if err := json.Unmarshal(body, &tx); err != nil {
		httpError(w, http.StatusBadRequest, "invalid tx json")
		return
	}

	// stateless validation
	if err := tx.ValidateBasic(); err != nil {
		httpError(w, http.StatusBadRequest, "tx validation failed: "+err.Error())
		return
	}

	// signature + from address
	from, err := types.RecoverTxSender(&tx)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid signature")
		return
	}

	// sanity check nonce / balance through mempool rules
	if err := s.node.Mempool.AddTx(&tx); err != nil {
		httpError(w, http.StatusBadRequest, "mempool reject: "+err.Error())
		return
	}

	// broadcast via p2p if available
	if s.node.P2P != nil {
		s.node.P2P.BroadcastTx(&tx)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "accepted",
		"hash":   tx.Hash().String(),
		"from":   from.String(),
	})
}

// -------------------- P2P ingress: tx/block --------------------

func (s *Server) handleP2PTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var tx types.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		httpError(w, http.StatusBadRequest, "invalid tx json")
		return
	}
	defer r.Body.Close()

	if err := tx.ValidateBasic(); err != nil {
		httpError(w, http.StatusBadRequest, "tx validation failed: "+err.Error())
		return
	}

	if _, err := types.RecoverTxSender(&tx); err != nil {
		httpError(w, http.StatusBadRequest, "invalid signature")
		return
	}

	// add but do not rebroadcast (avoid loops)
	if err := s.node.Mempool.AddTx(&tx); err != nil {
		httpError(w, http.StatusBadRequest, "mempool reject: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) handleP2PBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var b types.Block
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		httpError(w, http.StatusBadRequest, "invalid block json")
		return
	}
	defer r.Body.Close()

	// re-validate and execute via chain
	if err := s.node.Chain.AddBlock(&b); err != nil {
		httpError(w, http.StatusBadRequest, "block rejected: "+err.Error())
		return
	}

	// do not rebroadcast here; only original miner broadcasts
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"height": b.Header.Height,
		"hash":   b.Hash().String(),
	})
}

// -------------------- helpers --------------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		log.Printf("rpc: write json error: %v\n", err)
	}
}

func httpError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{
		"error": msg,
	})
}

func parseAddress(s string) (types.Address, error) {
	var zero types.Address

	s = strings.TrimSpace(s)
	if s == "" {
		return zero, errors.New("empty address")
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if len(s) != 40 {
		return zero, errors.New("invalid length")
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return zero, err
	}
	copy(zero[:], b)
	return zero, nil
}

// Optional helper to parse big-int from string if needed later.
func parseBigInt(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty")
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, errors.New("invalid big int")
	}
	return n, nil
}