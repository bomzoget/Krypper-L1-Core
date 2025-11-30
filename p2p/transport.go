// SPDX-License-Identifier: MIT
// Dev: KryperAI

package p2p

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Transport abstracts network I/O for P2P messages.
type Transport interface {
	PostJSON(peer *Peer, path string, payload any) error
}

type HTTPTransport struct {
	client *http.Client
}

func NewHTTPTransport() *HTTPTransport {
	return &HTTPTransport{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (t *HTTPTransport) PostJSON(peer *Peer, path string, payload any) error {
	if peer == nil {
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := peer.BaseURL + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("p2p: POST %s error: %v\n", url, err)
		return err
	}
	_ = resp.Body.Close()
	return nil
}