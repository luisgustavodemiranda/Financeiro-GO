// Package migrations aplica versões pendentes somente por comando explícito.
package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Apenas SQL da raiz: bootstrap administrativo não pertence a este executor.
//
//go:embed *.sql
var files embed.FS

type migration struct {
	version int
	sql     string
}
type record struct {
	version  int
	checksum string
}
type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func checksumSQL(sql string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ReplaceAll(sql, "\r\n", "\n"))))
}

func catalog() ([]migration, error) {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil, errors.New("não foi possível ler migrations incorporadas")
	}
	result := []migration{}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) < 6 || name[4] != '_' {
			return nil, errors.New("migration exige prefixo NNNN_")
		}
		version, err := strconv.Atoi(name[:4])
		if err != nil || version != len(result)+1 {
			return nil, errors.New("migrations devem ser consecutivas desde 0001")
		}
		sql, err := files.ReadFile(name)
		if err != nil || strings.TrimSpace(string(sql)) == "" {
			return nil, errors.New("migration vazia ou ilegível")
		}
		result = append(result, migration{version: version, sql: string(sql)})
	}
	return result, nil
}

// pending aceita somente um prefixo exato do histórico conhecido.
func pending(known []migration, applied []record) ([]migration, error) {
	if len(applied) > len(known) {
		return nil, errors.New("banco tem migrations desconhecidas por esta versão")
	}
	for i, r := range applied {
		if r.version != known[i].version || r.checksum != checksumSQL(known[i].sql) {
			return nil, errors.New("histórico de migrations divergente; não altere versões ou checksums manualmente")
		}
	}
	return known[len(applied):], nil
}

func history(ctx context.Context, q queryer) ([]record, error) {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname='financeiro')`).Scan(&exists); err != nil {
		return nil, errors.New("não foi possível inspecionar o schema")
	}
	if !exists {
		return []record{}, nil
	}
	rows, err := q.Query(ctx, `SELECT version,checksum FROM financeiro.schema_migrations ORDER BY version`)
	if err != nil {
		return nil, errors.New("schema existente sem histórico legível; revise antes de migrar")
	}
	defer rows.Close()
	result := []record{}
	for rows.Next() {
		var r record
		if err := rows.Scan(&r.version, &r.checksum); err != nil {
			return nil, errors.New("histórico de migrations inválido")
		}
		result = append(result, r)
	}
	if rows.Err() != nil || len(result) == 0 {
		return nil, errors.New("schema existente sem histórico completo; adoção automática recusada")
	}
	return result, nil
}

// Check não escreve. Exige todas as versões presentes no binário e hashes exatos.
func Check(ctx context.Context, q queryer) error {
	known, err := catalog()
	if err != nil {
		return err
	}
	applied, err := history(ctx, q)
	if err != nil {
		return err
	}
	next, err := pending(known, applied)
	if err != nil {
		return err
	}
	if len(next) > 0 {
		return fmt.Errorf("migration %04d pendente; execute cmd/migrate somente com autorização", next[0].version)
	}
	return nil
}

// Apply verifica todo o histórico antes de gravar e confirma o lote atomicamente.
func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	known, err := catalog()
	if err != nil {
		return err
	}
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
	applied, err := history(ctx, tx)
	if err != nil {
		return err
	}
	next, err := pending(known, applied)
	if err != nil {
		return err
	}
	for _, m := range next {
		if _, err := tx.Exec(ctx, m.sql); err != nil {
			return fmt.Errorf("migration %04d falhou; alterações do lote não foram confirmadas", m.version)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO financeiro.schema_migrations(version,checksum) VALUES($1,$2)`, m.version, checksumSQL(m.sql)); err != nil {
			return errors.New("não foi possível registrar a migration")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("não foi possível confirmar migrations; confira o estado antes de repetir")
	}
	return nil
}
