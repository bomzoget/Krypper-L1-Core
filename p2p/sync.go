// SPDX-License-Identifier: MIT
// Dev: KryperAI

package p2p

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"krypper-chain/types"
)

// SimpleSyncClient is a helper for pulling chain data from a single peer.
type SimpleSyncClient struct {
	baseURL string
	client  *http.Client
}

func NewSimpleSyncClient(peer *Peer) *SimpleSyncClient {
	if peer == nil {
		return nil
	}
	return &SimpleSyncClient{
		baseURL: peer.BaseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// FetchHead queries /chain/head on the remote node.
func (c *SimpleSyncClient) FetchHead() (*types.BlockHeader, error) {
	url := c.baseURL + "/chain/head"
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("remote error: %s", string(body))
	}

	var head struct {
		Height    uint64 `json:"height"`
		Hash      string `json:"hash"`
		StateRoot string `json:"stateRoot"`
		TxCount   int    `json:"txCount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&head); err != nil {
		return nil, err
	}

	// This is only a light representation; full header fetch
	// would require another dedicated endpoint.
	return &types.BlockHeader{
		Height: head.Height,
	}, nil
}

// FetchBlock is a placeholder for future extension.
// Expected pattern: GET /chain/block/{hash} on remote node.
func (c *SimpleSyncClient) FetchBlock(hash string) (*types.Block, error) {
	_ = hash
	return nil, fmt.Errorf("FetchBlock not implemented")
}