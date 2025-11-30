// SPDX-License-Identifier: MIT
// Dev: KryperAI

package p2p

// MessageType is a simple discriminator for future protocol extensions.
type MessageType string

const (
	MessageTypeTx    MessageType = "tx"
	MessageTypeBlock MessageType = "block"
)

// Envelope is a generic wrapper for P2P payloads.
type Envelope struct {
	Type MessageType `json:"type"`
	// Body is raw JSON of the underlying structure (tx or block).
	Body []byte `json:"body"`
}