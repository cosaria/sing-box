package protocol

import (
	"crypto/rand"
	"fmt"

	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

// Protocol defines the contract every proxy protocol must implement.
type Protocol interface {
	Name() string
	DisplayName() string
	DefaultSettings(port uint16) (string, error)
	BuildInbound(ib *store.Inbound) (option.Inbound, error)
	GenerateURL(ib *store.Inbound, host string) string
}

var registry = map[string]Protocol{}

// Register adds a protocol to the global registry.
func Register(p Protocol) {
	registry[p.Name()] = p
}

// Get returns the protocol for the given name, or nil if not found.
func Get(name string) Protocol {
	return registry[name]
}

// All returns all registered protocols.
func All() []Protocol {
	out := make([]Protocol, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}

// GenerateUUID creates a random UUID v4 string.
func GenerateUUID() string {
	var uuid [16]byte
	rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 1
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		uuid[0], uuid[1], uuid[2], uuid[3],
		uuid[4], uuid[5],
		uuid[6], uuid[7],
		uuid[8], uuid[9],
		uuid[10], uuid[11], uuid[12], uuid[13], uuid[14], uuid[15])
}

// GenerateShortID creates an 8-character hex string for Reality short_id.
func GenerateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%02x%02x%02x%02x", b[0], b[1], b[2], b[3])
}
