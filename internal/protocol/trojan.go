package protocol

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/cosaria/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

func init() {
	Register(&Trojan{})
}

type trojanSettings struct {
	Password string `json:"password"`
}

// Trojan 实现 Trojan 协议。
type Trojan struct{}

func (tr *Trojan) Name() string        { return "trojan" }
func (tr *Trojan) DisplayName() string { return "Trojan" }

func (tr *Trojan) DefaultSettings(port uint16) (string, error) {
	settings := trojanSettings{Password: GenerateUUID()}
	b, err := json.Marshal(settings)
	return string(b), err
}

func (tr *Trojan) BuildInbound(ib *store.Inbound) (option.Inbound, error) {
	var s trojanSettings
	if err := json.Unmarshal([]byte(ib.Settings), &s); err != nil {
		return option.Inbound{}, fmt.Errorf("invalid trojan settings: %w", err)
	}
	return option.Inbound{
		Type: "trojan",
		Tag:  ib.Tag,
		Options: &option.TrojanInboundOptions{
			ListenOptions: listenAll(ib.Port),
			Users: []option.TrojanUser{
				{Name: "default", Password: s.Password},
			},
		},
	}, nil
}

func (tr *Trojan) GenerateURL(ib *store.Inbound, host string) string {
	var s trojanSettings
	if err := json.Unmarshal([]byte(ib.Settings), &s); err != nil {
		return ""
	}
	return fmt.Sprintf("trojan://%s@%s:%d#%s",
		url.PathEscape(s.Password), host, ib.Port, url.PathEscape(ib.Tag))
}
