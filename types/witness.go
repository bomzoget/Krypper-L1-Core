// SPDX-License-Identifier: MIT
// Dev: KryperAI

package types

// Witness represents a tier-3 mobile miner attestation for a block header.
type Witness struct {
	BlockHeight uint64  `json:"height"`  // height being witnessed
	Address     Address `json:"address"` // mobile miner address
	Signature   []byte  `json:"signature"`
	Hash        Hash    `json:"hash"` // block header hash that was signed
}