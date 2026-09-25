package server

import (
	"encoding/json"
	"financeirogo/internal/cartoes"
	"financeirogo/internal/config"
	"financeirogo/internal/contas"
	"net/http"
)

func NewHandler() http.Handler {
	return NewHandlerWithRepositories(contas.NewMemoryRepository(), cartoes.NewMemoryRepository(), "memoria; dados perdidos ao reiniciar")
}

func NewHandlerWithRepositories(repo contas.Repository, cards cartoes.Repository, persistence string) http.Handler {
	mux := http.NewServeMux()
	registerAccountRoutes(mux, contas.NewService(repo))
	registerCardRoutes(mux, cartoes.NewService(cards))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"application": "Financeiro-GO", "phase": "contas e lancamentos", "persistence": persistence})
	})
	return mux
}

func New(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{Addr: cfg.Address, Handler: handler, ReadTimeout: cfg.ReadTimeout, ReadHeaderTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout, MaxHeaderBytes: 1 << 20}
}
