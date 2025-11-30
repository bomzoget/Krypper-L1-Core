// SPDX-License-Identifier: MIT
// Dev: KryperAI

package rpc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"strings"

	"krypper-chain/node"
	"krypper-chain/types"
)

type Server struct {
	Node *node.Node
}

func NewServer(n *node.Node) *Server {
	return &Server{Node: n}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/tx/send", s.handleSendTx)
	mux.HandleFunc("/account/", s.handleAccount)
	mux.HandleFunc("/chain/head", s.handleHead)
	mux.HandleFunc("/mempool/info", s.handleMempoolInfo)

	log.Printf("RPC listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}

// ------------------------------------------------------------------
// Models
// ------------------------------------------------------------------

type sendTxRequest struct {
	ChainID  string `json:"chainId"`
	Nonce    uint64 `json:"nonce"`
	To       string `json:"to"`
	Value    string `json:"value"`
	GasPrice string `json:"gasPrice"`
	GasLimit uint64 `json:"gasLimit"`
	Data     string `json:"data"`
	R        string `json:"r"`
	S        string `json:"s"`
	V        uint8  `json:"v"`
}

type sendTxResponse struct {
	TxHash string `json:"txHash"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type accountResponse struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

// ------------------------------------------------------------------
// Handlers
// ------------------------------------------------------------------

func (s *Server) handleSendTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sendTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	chainID, err := parseBig(req.ChainID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chainId")
		return
	}

	to, err := parseAddress(req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to address")
		return
	}

	value, err := parseBig(req.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid value")
		return
	}

	gasPrice, err := parseBig(req.GasPrice)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid gasPrice")
		return
	}

	data, err := parseHexBytes(req.Data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid data hex")
		return
	}

	rBig, err := parseBigHex(req.R)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid r")
		return
	}

	sBig, err := parseBigHex(req.S)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid s")
		return
	}

	tx := &types.Transaction{
		ChainId:  chainID,
		Type:     types.TxTypeTransfer,
		Nonce:    req.Nonce,
		To:       to,
		Value:    value,
		GasPrice: gasPrice,
		GasLimit: req.GasLimit,
		Data:     data,
		Signature: types.Signature{
			R: rBig,
			S: sBig,
			V: req.V,
		},
	}

	if err := tx.ValidateBasic(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid tx: "+err.Error())
		return
	}

	if err := s.Node.Mempool.AddTx(tx); err != nil {
		writeJSON(w, http.StatusOK, sendTxResponse{
			TxHash: tx.Hash().String(),
			Status: "rejected",
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, sendTxResponse{
		TxHash: tx.Hash().String(),
		Status: "accepted",
	})
}

func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// URL: /account/0x...
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/account/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "missing address")
		return
	}

	addr, err := parseAddress(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid address")
		return
	}

	acc := s.Node.State.GetAccount(addr)

	resp := accountResponse{
		Address: addr.String(),
		Balance: acc.Balance.String(),
		Nonce:   acc.Nonce,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	head := s.Node.Chain.Head()
	if head == nil {
		writeError(w, http.StatusNotFound, "no head")
		return
	}

	type headResponse struct {
		Height    uint64 `json:"height"`
		Hash      string `json:"hash"`
		StateRoot string `json:"stateRoot"`
		TxCount   int    `json:"txCount"`
	}

	resp := headResponse{
		Height:    head.Header.Height,
		Hash:      head.Hash().String(),
		StateRoot: head.Header.StateRoot.String(),
		TxCount:   len(head.Transactions),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleMempoolInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	type memInfo struct {
		Pending int `json:"pending"`
	}

	resp := memInfo{
		Pending: s.Node.Mempool.Count(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{
		"error": msg,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func parseAddress(s string) (types.Address, error) {
	var a types.Address
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	if len(s) != 40 {
		return a, errors.New("invalid length")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return a, err
	}
	copy(a[:], b)
	return a, nil
}

func parseBig(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, errors.New("invalid big int")
	}
	return n, nil
}

func parseHexBytes(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	s = strings.TrimPrefix(s, "0x")
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

func parseBigHex(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0), nil
	}
	s = strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(b), nil
}