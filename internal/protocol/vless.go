package protocol

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/cosaria/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

func init() {
	Register(&VLESS{})
}

type vlessRealityHandshake struct {
	Server     string `json:"server"`
	ServerPort uint16 `json:"server_port"`
}

type vlessReality struct {
	PrivateKey string                `json:"private_key"`
	PublicKey  string                `json:"public_key"`
	ShortID    string                `json:"short_id"`
	Handshake  vlessRealityHandshake `json:"handshake"`
}

type vlessSettings struct {
	UUID    string       `json:"uuid"`
	Flow    string       `json:"flow"`
	Reality vlessReality `json:"reality"`
}

// VLESS 实现 VLESS-REALITY 协议。
type VLESS struct{}

func (v *VLESS) Name() string        { return "vless" }
func (v *VLESS) DisplayName() string { return "VLESS-REALITY" }

func (v *VLESS) DefaultSettings(port uint16) (string, error) {
	privateKey, publicKey, err := generateX25519KeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate X25519 keypair: %w", err)
	}
	settings := vlessSettings{
		UUID: GenerateUUID(),
		Flow: "xtls-rprx-vision",
		Reality: vlessReality{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
			ShortID:    GenerateShortID(),
			Handshake: vlessRealityHandshake{
				Server:     "www.microsoft.com",
				ServerPort: 443,
			},
		},
	}
	b, err := json.Marshal(settings)
	return string(b), err
}

func (v *VLESS) BuildInbound(ib *store.Inbound) (option.Inbound, error) {
	var s vlessSettings
	if err := json.Unmarshal([]byte(ib.Settings), &s); err != nil {
		return option.Inbound{}, fmt.Errorf("invalid vless settings: %w", err)
	}
	return option.Inbound{
		Type: "vless",
		Tag:  ib.Tag,
		Options: &option.VLESSInboundOptions{
			ListenOptions: option.ListenOptions{
				ListenPort: ib.Port,
			},
			Users: []option.VLESSUser{
				{Name: "default", UUID: s.UUID, Flow: s.Flow},
			},
			InboundTLSOptionsContainer: option.InboundTLSOptionsContainer{
				TLS: &option.InboundTLSOptions{
					Enabled:    true,
					ServerName: s.Reality.Handshake.Server,
					Reality: &option.InboundRealityOptions{
						Enabled: true,
						Handshake: option.InboundRealityHandshakeOptions{
							ServerOptions: option.ServerOptions{
								Server:     s.Reality.Handshake.Server,
								ServerPort: s.Reality.Handshake.ServerPort,
							},
						},
						PrivateKey: s.Reality.PrivateKey,
						ShortID:    badoption.Listable[string]{s.Reality.ShortID},
					},
				},
			},
		},
	}, nil
}

func (v *VLESS) GenerateURL(ib *store.Inbound, host string) string {
	var s vlessSettings
	if err := json.Unmarshal([]byte(ib.Settings), &s); err != nil {
		return ""
	}
	params := url.Values{}
	params.Set("type", "tcp")
	params.Set("security", "reality")
	params.Set("sni", s.Reality.Handshake.Server)
	params.Set("fp", "chrome")
	params.Set("pbk", s.Reality.PublicKey)
	params.Set("sid", s.Reality.ShortID)
	params.Set("flow", s.Flow)
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		s.UUID, host, ib.Port, params.Encode(), url.PathEscape(ib.Tag))
}

func generateX25519KeyPair() (privateKeyStr, publicKeyStr string, err error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	privateKeyStr = base64.RawURLEncoding.EncodeToString(key.Bytes())
	publicKeyStr = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	return privateKeyStr, publicKeyStr, nil
}
