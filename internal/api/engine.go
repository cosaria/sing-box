package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := "stopped"
	if s.engine.Running() {
		status = "running"
	}

	inbounds, _ := s.store.ListInbounds()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
		"engine":   status,
		"inbounds": len(inbounds),
	})
}

func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.Reload(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"}) //nolint:errcheck
}

func (s *Server) handleEngineStart(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.Start(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"}) //nolint:errcheck
}

func (s *Server) handleEngineStop(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.Stop(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"}) //nolint:errcheck
}
