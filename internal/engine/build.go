package engine

import (
	"fmt"

	"github.com/233boy/sing-box/internal/protocol"
	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

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
	p := protocol.Get(ib.Protocol)
	if p == nil {
		return option.Inbound{}, fmt.Errorf("unsupported protocol: %s", ib.Protocol)
	}
	return p.BuildInbound(ib)
}
