package api

import (
	"net/http"

	"github.com/233boy/sing-box/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	summaries, err := s.store.GetTrafficSummary()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if summaries == nil {
		summaries = []store.TrafficSummary{}
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (s *Server) handleGetStatsByTag(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	logs, err := s.store.GetTrafficByTag(tag)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if logs == nil {
		logs = []store.TrafficLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}
