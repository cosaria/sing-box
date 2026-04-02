package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/233boy/sing-box/internal/protocol"
	"github.com/233boy/sing-box/internal/store"
	"github.com/go-chi/chi/v5"
)

type createInboundRequest struct {
	Protocol string          `json:"protocol"`
	Port     uint16          `json:"port"`
	Settings json.RawMessage `json:"settings"`
}

type updateInboundRequest struct {
	Port     *uint16          `json:"port,omitempty"`
	Settings *json.RawMessage `json:"settings,omitempty"`
}

func (s *Server) handleListInbounds(w http.ResponseWriter, r *http.Request) {
	inbounds, err := s.store.ListInbounds()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if inbounds == nil {
		inbounds = []*store.Inbound{}
	}
	writeJSON(w, http.StatusOK, inbounds)
}

func (s *Server) handleCreateInbound(w http.ResponseWriter, r *http.Request) {
	var req createInboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Protocol == "" {
		writeError(w, http.StatusBadRequest, "protocol is required")
		return
	}
	if req.Port == 0 {
		writeError(w, http.StatusBadRequest, "port is required")
		return
	}

	p := protocol.Get(req.Protocol)
	if p == nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported protocol: %s", req.Protocol))
		return
	}

	var settingsStr string
	if req.Settings == nil || string(req.Settings) == "{}" || string(req.Settings) == "null" {
		generated, err := p.DefaultSettings(req.Port)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate settings: "+err.Error())
			return
		}
		settingsStr = generated
	} else {
		settingsStr = string(req.Settings)
	}

	tag := fmt.Sprintf("%s-%d", req.Protocol, req.Port)

	ib := &store.Inbound{
		Tag:      tag,
		Protocol: req.Protocol,
		Port:     req.Port,
		Settings: settingsStr,
	}

	if err := s.store.CreateInbound(ib); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	s.engine.Reload() //nolint:errcheck
	writeJSON(w, http.StatusCreated, ib)
}

func (s *Server) handleGetInbound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ib, err := s.store.GetInbound(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "inbound not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ib)
}

func (s *Server) handleUpdateInbound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ib, err := s.store.GetInbound(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "inbound not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req updateInboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Port != nil {
		ib.Port = *req.Port
	}
	if req.Settings != nil {
		ib.Settings = string(*req.Settings)
	}

	if err := s.store.UpdateInbound(ib); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.engine.Reload() //nolint:errcheck
	writeJSON(w, http.StatusOK, ib)
}

func (s *Server) handleDeleteInbound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := s.store.DeleteInbound(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "inbound not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.engine.Reload() //nolint:errcheck
	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}
