package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/cosaria/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

func init() {
	Register(&Shadowsocks{})
}

type shadowsocksSettings struct {
	Method   string `json:"method"`
	Password string `json:"password"`
}

// Shadowsocks 实现 Shadowsocks 2022 协议。
type Shadowsocks struct{}

func (s *Shadowsocks) Name() string        { return "shadowsocks" }
func (s *Shadowsocks) DisplayName() string { return "Shadowsocks" }

func (s *Shadowsocks) DefaultSettings(port uint16) (string, error) {
	settings := shadowsocksSettings{
		Method:   "2022-blake3-aes-128-gcm",
		Password: GenerateUUID(),
	}
	b, err := json.Marshal(settings)
	return string(b), err
}

func (s *Shadowsocks) BuildInbound(ib *store.Inbound) (option.Inbound, error) {
	var ss shadowsocksSettings
	if err := json.Unmarshal([]byte(ib.Settings), &ss); err != nil {
		return option.Inbound{}, fmt.Errorf("invalid shadowsocks settings: %w", err)
	}

	password := ss.Password
	if isSSAEAD2022(ss.Method) {
		password = uuidToBase64Key(ss.Password)
	}

	return option.Inbound{
		Type: "shadowsocks",
		Tag:  ib.Tag,
		Options: &option.ShadowsocksInboundOptions{
			ListenOptions: option.ListenOptions{
				ListenPort: ib.Port,
			},
			Method:   ss.Method,
			Password: password,
		},
	}, nil
}

func (s *Shadowsocks) GenerateURL(ib *store.Inbound, host string) string {
	var ss shadowsocksSettings
	if err := json.Unmarshal([]byte(ib.Settings), &ss); err != nil {
		return ""
	}

	password := ss.Password
	if isSSAEAD2022(ss.Method) {
		password = uuidToBase64Key(ss.Password)
	}

	userInfo := base64.StdEncoding.EncodeToString([]byte(ss.Method + ":" + password))
	return fmt.Sprintf("ss://%s@%s:%d#%s", userInfo, host, ib.Port, url.PathEscape(ib.Tag))
}

func isSSAEAD2022(method string) bool {
	switch method {
	case "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305":
		return true
	}
	return false
}

func uuidToBase64Key(uuidStr string) string {
	clean := ""
	for _, c := range uuidStr {
		if c != '-' {
			clean += string(c)
		}
	}
	if len(clean) != 32 {
		return uuidStr
	}
	var raw [16]byte
	for i := 0; i < 16; i++ {
		fmt.Sscanf(clean[i*2:i*2+2], "%02x", &raw[i])
	}
	return base64.StdEncoding.EncodeToString(raw[:])
}
