package main

import (
	"context"
	"errors"
	"financeirogo/internal/cartoes"
	"financeirogo/internal/config"
	"financeirogo/internal/contas"
	"financeirogo/internal/database"
	"financeirogo/internal/server"
	"financeirogo/migrations"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var repo contas.Repository = contas.NewMemoryRepository()
	var cards cartoes.Repository = cartoes.NewMemoryRepository()
	persistence := "memoria; dados perdidos ao reiniciar"
	if cfg.Persistence == "postgres" {
		startup, cancel := context.WithTimeout(ctx, cfg.DatabaseTimeout)
		pool, err := database.Open(startup, cfg.DatabaseURL)
		if err != nil {
			cancel()
			return err
		}
		defer pool.Close()
		err = migrations.Check(startup, pool)
		cancel()
		if err != nil {
			return err
		}
		repo = contas.NewPostgresRepository(pool, cfg.DatabaseTimeout)
		cards = cartoes.NewPostgresRepository(pool, cfg.DatabaseTimeout)
		persistence = "postgresql"
	}
	srv := server.New(cfg, server.NewHandlerWithRepositories(repo, cards, persistence))
	done := make(chan error, 1)
	go func() {
		log.Printf("Financeiro-GO disponível em http://%s", cfg.Address)
		done <- srv.ListenAndServe()
	}()
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			_ = srv.Close()
			return err
		}
		err := <-done
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
func main() {
	if err := run(); err != nil {
		log.Printf("falha ao executar aplicação: %v", err)
		os.Exit(1)
	}
}
