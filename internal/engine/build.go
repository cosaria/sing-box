package engine

import (
	"encoding/json"
	"fmt"

	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

type shadowsocksSettings struct {
	Method   string `json:"method"`
	Password string `json:"password"`
}

func buildOptions(inbounds []*store.Inbound) (option.Options, error) {
	opts := option.Options{
		Log: &option.LogOptions{
			Level: "info",
		},
		Outbounds: []option.Outbound{
			{Type: "direct", Tag: "direct"},
		},
	}

	for _, ib := range inbounds {
		singIb, err := buildInbound(ib)
		if err != nil {
			return opts, fmt.Errorf("failed to build inbound %q: %w", ib.Tag, err)
		}
		opts.Inbounds = append(opts.Inbounds, singIb)
	}
	return opts, nil
}

func buildInbound(ib *store.Inbound) (option.Inbound, error) {
	switch ib.Protocol {
	case "shadowsocks":
		return buildShadowsocks(ib)
	default:
		return option.Inbound{}, fmt.Errorf("unsupported protocol: %s", ib.Protocol)
	}
}

func buildShadowsocks(ib *store.Inbound) (option.Inbound, error) {
	var ss shadowsocksSettings
	if err := json.Unmarshal([]byte(ib.Settings), &ss); err != nil {
		return option.Inbound{}, fmt.Errorf("invalid shadowsocks settings: %w", err)
	}

	return option.Inbound{
		Type: "shadowsocks",
		Tag:  ib.Tag,
		Options: &option.ShadowsocksInboundOptions{
			ListenOptions: option.ListenOptions{
				ListenPort: ib.Port,
			},
			Method:   ss.Method,
			Password: ss.Password,
		},
	}, nil
}
