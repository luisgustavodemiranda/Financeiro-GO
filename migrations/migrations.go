// Package migrations aplica somente migrations explícitas, nunca na inicialização HTTP.
package migrations

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 0001_accounts_entries.sql
var initialSQL string

func checksum() string {
	// O mesmo arquivo deve ter o mesmo hash no Windows (CRLF) e Linux (LF).
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ReplaceAll(initialSQL, "\r\n", "\n"))))
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Check é somente leitura. Um schema desconhecido não é adotado nem modificado.
func Check(ctx context.Context, q queryer) error {
	var version, count int
	var hash string
	err := q.QueryRow(ctx, `SELECT version, checksum, (SELECT count(*) FROM financeiro.schema_migrations)
 FROM financeiro.schema_migrations WHERE version=1`).Scan(&version, &hash, &count)
	if err != nil || version != 1 || count != 1 || hash != checksum() {
		return errors.New("migration 0001 ausente ou incompatível; revise o banco e execute cmd/migrate somente com autorização")
	}
	return nil
}

// Apply deve ser chamado somente por comando explicitamente autorizado.
func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return errors.New("não foi possível iniciar a migration")
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	var name string
	if err := tx.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil || (name != "financeiro_go" && name != "financeiro_go_test") {
		return errors.New("migration permitida somente nos bancos exclusivos do Financeiro-GO")
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(736204103)`); err != nil {
		return errors.New("não foi possível obter o bloqueio da migration")
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname='financeiro')`).Scan(&exists); err != nil {
		return errors.New("não foi possível inspecionar o schema")
	}
	if exists {
		if err := Check(ctx, tx); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, initialSQL); err != nil {
			return errors.New("migration 0001 falhou; transação revertida; confira permissões e compatibilidade do banco")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO financeiro.schema_migrations(version, checksum) VALUES(1,$1)`, checksum()); err != nil {
			return errors.New("não foi possível registrar a migration")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("não foi possível confirmar a migration; confira seu estado antes de repetir")
	}
	return nil
}
