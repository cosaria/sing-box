package api

import (
	"context"
	"net"
	"net/http"

	"github.com/cosaria/sing-box/internal/store"
	"github.com/go-chi/chi/v5"
)

type EngineController interface {
	Start() error
	Stop() error
	Reload() error
	Running() bool
}

type Server struct {
	engine   EngineController
	store    *store.Store
	router   chi.Router
	httpSrv  *http.Server
	token    string
	subToken string
}

func NewServer(engine EngineController, st *store.Store, svc any, listenAddr, token, subToken string) *Server {
	s := &Server{
		engine:   engine,
		store:    st,
		token:    token,
		subToken: subToken,
	}

	r := chi.NewRouter()

	// Public routes (no auth)
	r.Get("/sub/{token}", s.handleSubscription)

	// Authenticated API routes
	r.Group(func(r chi.Router) {
		r.Use(tokenAuth(token))
		r.Get("/api/status", s.handleStatus)
		r.Post("/api/reload", s.handleReload)
		r.Post("/api/engine/start", s.handleEngineStart)
		r.Post("/api/engine/stop", s.handleEngineStop)
		r.Get("/api/inbounds", s.handleListInbounds)
		r.Post("/api/inbounds", s.handleCreateInbound)
		r.Get("/api/inbounds/{id}", s.handleGetInbound)
		r.Put("/api/inbounds/{id}", s.handleUpdateInbound)
		r.Delete("/api/inbounds/{id}", s.handleDeleteInbound)
		r.Get("/api/stats", s.handleGetStats)
		r.Get("/api/stats/{tag}", s.handleGetStatsByTag)
	})

	s.router = r
	s.httpSrv = &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}
	return s
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		return err
	}
	go s.httpSrv.Serve(ln) //nolint:errcheck
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}
