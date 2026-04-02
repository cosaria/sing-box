package api

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/233boy/sing-box/internal/protocol"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleSubscription(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token != s.subToken {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	inbounds, err := s.store.ListInbounds()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	host := r.Host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	var urls []string
	for _, ib := range inbounds {
		p := protocol.Get(ib.Protocol)
		if p == nil {
			continue
		}
		u := p.GenerateURL(ib, host)
		if u != "" {
			urls = append(urls, u)
		}
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(strings.Join(urls, "\n")))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(encoded)) //nolint:errcheck
}
