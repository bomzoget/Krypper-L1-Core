// SPDX-License-Identifier: MIT
// Dev: KryperAI

package p2p

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"krypper-chain/types"
)

// Manager is a very simple HTTP-based gossip layer.
// It sends tx and blocks to known peer RPC endpoints.
type Manager struct {
	peers  []string
	client *http.Client
}

func NewManager(peers []string) *Manager {
	norm := make([]string, 0, len(peers))
	for _, p := range peers {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "http://") && !strings.HasPrefix(p, "https://") {
			p = "http://" + p
		}
		p = strings.TrimRight(p, "/")
		norm = append(norm, p)
	}
	return &Manager{
		peers:  norm,
		client: &http.Client{},
	}
}

// BroadcastTx sends raw transaction to peers.
// Uses /p2p/tx endpoint on remote nodes.
func (m *Manager) BroadcastTx(tx *types.Transaction) {
	if tx == nil || len(m.peers) == 0 {
		return
	}

	payload, err := json.Marshal(tx)
	if err != nil {
		log.Printf("p2p: marshal tx error: %v\n", err)
		return
	}

	for _, peer := range m.peers {
		url := peer + "/p2p/tx"
		go m.post(url, payload)
	}
}

// BroadcastBlock sends full block json to peers.
// Uses /p2p/block endpoint on remote nodes.
func (m *Manager) BroadcastBlock(b *types.Block) {
	if b == nil || len(m.peers) == 0 {
		return
	}

	payload, err := json.Marshal(b)
	if err != nil {
		log.Printf("p2p: marshal block error: %v\n", err)
		return
	}

	for _, peer := range m.peers {
		url := peer + "/p2p/block"
		go m.post(url, payload)
	}
}

func (m *Manager) post(url string, body []byte) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("p2p: build request error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		log.Printf("p2p: post %s error: %v\n", url, err)
		return
	}
	_ = resp.Body.Close()
}