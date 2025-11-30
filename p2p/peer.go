// SPDX-License-Identifier: MIT
// Dev: KryperAI

package p2p

import "strings"

// Peer represents a remote node reachable over HTTP.
type Peer struct {
	BaseURL string
}

func NewPeer(raw string) *Peer {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "http://" + raw
	}

	raw = strings.TrimRight(raw, "/")

	return &Peer{BaseURL: raw}
}