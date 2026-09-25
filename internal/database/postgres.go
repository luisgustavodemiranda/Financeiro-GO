// Package database configura exclusivamente conexões do Financeiro-GO.
package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ParseConfig não retorna erros do driver: eles podem conter a URL e sua senha.
func ParseConfig(raw string) (*pgxpool.Config, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("FINANCEIRO_DATABASE_URL é obrigatória")
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		return nil, errors.New("FINANCEIRO_DATABASE_URL inválida")
	}
	if cfg.ConnConfig.Database != "financeiro_go" && cfg.ConnConfig.Database != "financeiro_go_test" {
		return nil, errors.New("banco deve ser financeiro_go ou financeiro_go_test; não use o banco do Fiscal-GO")
	}
	cfg.MaxConns = 5
	cfg.MinConns = 0
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.ConnConfig.RuntimeParams["application_name"] = "financeiro-go"
	return cfg, nil
}

func Open(ctx context.Context, raw string) (*pgxpool.Pool, error) {
	cfg, err := ParseConfig(raw)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("não foi possível configurar o pool PostgreSQL")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.New("não foi possível conectar ao PostgreSQL; confira configuração e disponibilidade")
	}
	var name string
	var superuser bool
	if err := pool.QueryRow(ctx, `SELECT current_database(), rolsuper FROM pg_catalog.pg_roles WHERE rolname=current_user`).Scan(&name, &superuser); err != nil || name != cfg.ConnConfig.Database || superuser {
		pool.Close()
		return nil, errors.New("conexão exige banco exclusivo e usuário sem superusuário")
	}
	return pool, nil
}
