// SPDX-License-Identifier: MIT
// Dev: KryperAI

package p2p

import (
	"log"

	"krypper-chain/types"
)

// Manager is a simple high-level P2P controller for broadcasting
// transactions and blocks to known peers.
type Manager struct {
	peers     []*Peer
	transport Transport
}

func NewManager(addrs []string) *Manager {
	peers := make([]*Peer, 0, len(addrs))
	for _, raw := range addrs {
		p := NewPeer(raw)
		if p != nil {
			peers = append(peers, p)
		}
	}
	return &Manager{
		peers:     peers,
		transport: NewHTTPTransport(),
	}
}

// Peers returns a copy of the internal peer list.
func (m *Manager) Peers() []*Peer {
	out := make([]*Peer, 0, len(m.peers))
	out = append(out, m.peers...)
	return out
}

// AddPeer appends a new peer to the manager.
func (m *Manager) AddPeer(addr string) {
	p := NewPeer(addr)
	if p == nil {
		return
	}
	m.peers = append(m.peers, p)
}

// BroadcastTx sends a transaction to all known peers.
// Remote nodes are expected to expose /p2p/tx endpoint.
func (m *Manager) BroadcastTx(tx *types.Transaction) {
	if tx == nil || len(m.peers) == 0 {
		return
	}
	for _, p := range m.peers {
		go func(peer *Peer) {
			if err := m.transport.PostJSON(peer, "/p2p/tx", tx); err != nil {
				log.Printf("p2p: broadcast tx to %s failed: %v\n", peer.BaseURL, err)
			}
		}(p)
	}
}

// BroadcastBlock sends a block to all known peers.
// Remote nodes are expected to expose /p2p/block endpoint.
func (m *Manager) BroadcastBlock(b *types.Block) {
	if b == nil || len(m.peers) == 0 {
		return
	}
	for _, p := range m.peers {
		go func(peer *Peer) {
			if err := m.transport.PostJSON(peer, "/p2p/block", b); err != nil {
				log.Printf("p2p: broadcast block to %s failed: %v\n", peer.BaseURL, err)
			}
		}(p)
	}
}