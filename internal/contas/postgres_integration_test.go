package contas_test

import (
	"context"
	"encoding/json"
	"errors"
	"financeirogo/internal/contas"
	"financeirogo/internal/database"
	"financeirogo/internal/server"
	"financeirogo/migrations"
	"math"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPostgresIntegration não cria banco/schema nem aplica migrations.
// O opt-in permite gravar e remover apenas os registros fictícios desta execução.
func TestPostgresIntegration(t *testing.T) {
	url := os.Getenv("FINANCEIRO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PostgreSQL não executado: configure FINANCEIRO_TEST_DATABASE_URL em financeiro_go_test já migrado")
	}
	cfg, err := database.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "financeiro_go_test" {
		t.Fatal("integração exige financeiro_go_test; banco da aplicação não é permitido")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	open := func(t *testing.T) *pgxpool.Pool {
		t.Helper()
		pool, err := database.Open(ctx, url)
		if err != nil {
			t.Fatal(err)
		}
		return pool
	}
	pool := open(t)
	defer pool.Close()
	if err := migrations.Check(ctx, pool); err != nil {
		t.Fatal(err)
	}
	otherPool := open(t)
	defer otherPool.Close()
	s := contas.NewService(contas.NewPostgresRepository(pool, 5*time.Second))
	other := contas.NewService(contas.NewPostgresRepository(otherPool, 5*time.Second))
	ids := []string{}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		tx, err := pool.Begin(cleanup)
		if err != nil {
			t.Error("não foi possível iniciar limpeza das fixtures")
			return
		}
		defer tx.Rollback(cleanup)
		if _, err := tx.Exec(cleanup, `DELETE FROM financeiro.entries WHERE account_id::text=ANY($1::text[])`, ids); err != nil {
			t.Error("falha ao limpar lançamentos fictícios")
			return
		}
		if _, err := tx.Exec(cleanup, `DELETE FROM financeiro.accounts WHERE id::text=ANY($1::text[])`, ids); err != nil {
			t.Error("falha ao limpar contas fictícias")
			return
		}
		if err := tx.Commit(cleanup); err != nil {
			t.Error("falha ao confirmar limpeza das fixtures")
		}
	}()
	create := func(t *testing.T, initial int64) contas.Account {
		t.Helper()
		a, err := s.Create(ctx, "Conta fictícia de integração", initial, "2026-01-01")
		if err != nil {
			t.Fatal("falha ao criar fixture")
		}
		ids = append(ids, a.ID)
		return a
	}
	get := func(t *testing.T, id string) contas.State {
		t.Helper()
		state, err := s.Get(ctx, id)
		if err != nil {
			t.Fatal("falha ao consultar fixture")
		}
		return state
	}

	t.Run("persistencia HTTP e novo pool", func(t *testing.T) {
		a := create(t, 1000)
		if len(get(t, a.ID).Entries) != 0 {
			t.Fatal("saldo inicial gerou receita")
		}
		h := server.NewHandlerWithRepository(contas.NewPostgresRepository(pool, 5*time.Second), "postgresql")
		for _, body := range []string{
			`{"kind":"income","description":"Receita ficticia","amount_cents":500,"date":"2026-01-01"}`,
			`{"kind":"expense","description":"Despesa ficticia","amount_cents":1800,"date":"2026-01-02"}`,
		} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/accounts/"+a.ID+"/entries", strings.NewReader(body)))
			if w.Code != 201 {
				t.Fatalf("registro HTTP retornou %d", w.Code)
			}
		}
		// Nova conexão e novo handler devem enxergar registros já confirmados.
		reopened := open(t)
		defer reopened.Close()
		h = server.NewHandlerWithRepository(contas.NewPostgresRepository(reopened, 5*time.Second), "postgresql")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/accounts/"+a.ID+"/balance", nil))
		var result contas.Account
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || result.BalanceCents != -300 || result.InitialBalanceCents != 1000 {
			t.Fatal("saldo não persistiu corretamente")
		}
		state := get(t, a.ID)
		if len(state.Entries) != 2 || state.Entries[0].AmountCents != 500 || state.Entries[1].Kind != "expense" {
			t.Fatal("histórico inconsistente")
		}
		list, err := other.List(ctx)
		if err != nil {
			t.Fatal("falha na listagem")
		}
		found := false
		for _, item := range list {
			if item.ID == a.ID && item.BalanceCents == -300 {
				found = true
			}
		}
		if !found {
			t.Fatal("conta ausente da listagem")
		}
		isolated := create(t, 0)
		if get(t, isolated.ID).Account.BalanceCents != 0 {
			t.Fatal("contas não isoladas")
		}
	})

	t.Run("limites e rollback", func(t *testing.T) {
		for _, tc := range []struct {
			initial int64
			kind    string
		}{{math.MaxInt64 - 1, "income"}, {math.MinInt64 + 1, "expense"}} {
			a := create(t, tc.initial)
			if _, err := s.Register(ctx, a.ID, tc.kind, "Limite fictício", 1, "2026-01-01"); err != nil {
				t.Fatal("limite válido rejeitado")
			}
			before := get(t, a.ID)
			if _, err := s.Register(ctx, a.ID, tc.kind, "Overflow fictício", 1, "2026-01-01"); !errors.Is(err, contas.ErrOverflow) {
				t.Fatal("overflow não rejeitado")
			}
			after := get(t, a.ID)
			if len(after.Entries) != 1 || after.Account != before.Account {
				t.Fatal("erro alterou estado")
			}
		}
		a := create(t, 123)
		repo := contas.NewPostgresRepository(pool, 5*time.Second)
		err := repo.Update(ctx, a.ID, func(state *contas.State) error {
			state.Account.BalanceCents = 999
			return contas.ErrInvalid
		})
		if !errors.Is(err, contas.ErrInvalid) || get(t, a.ID).Account.BalanceCents != 123 {
			t.Fatal("callback com erro não foi revertido")
		}
		// Força erro SQL de CHECK; a transação não pode alterar o saldo.
		err = repo.Update(ctx, a.ID, func(state *contas.State) error {
			state.Account.BalanceCents = 124
			state.Entries = append(state.Entries, contas.Entry{ID: a.ID + "-1", AccountID: a.ID, Kind: "invalid", Description: "Fixture", AmountCents: 1, Date: "2026-01-01"})
			return nil
		})
		if err == nil {
			t.Fatal("constraint não rejeitou lançamento")
		}
		state := get(t, a.ID)
		if state.Account.BalanceCents != 123 || len(state.Entries) != 0 {
			t.Fatal("falha SQL gravou estado parcial")
		}
		if _, err := s.Register(ctx, a.ID, "income", "Fixture", 1, "2026-01-01"); err != nil {
			t.Fatal("conexão não recuperou após rollback")
		}
	})

	t.Run("concorrencia entre pools e snapshot", func(t *testing.T) {
		a := create(t, 0)
		var wg sync.WaitGroup
		for i := range 24 {
			service := s
			if i%2 == 0 {
				service = other
			}
			wg.Go(func() {
				if _, err := service.Register(ctx, a.ID, "income", "Fixture concorrente", 1, "2026-01-01"); err != nil {
					t.Error("registro concorrente falhou")
				}
			})
		}
		for range 24 {
			state, err := other.Get(ctx, a.ID)
			if err != nil {
				t.Error("leitura concorrente falhou")
				break
			}
			if state.Account.BalanceCents != int64(len(state.Entries)) {
				t.Error("saldo e histórico de snapshots diferentes")
			}
		}
		wg.Wait()
		state := get(t, a.ID)
		if state.Account.BalanceCents != 24 || len(state.Entries) != 24 {
			t.Fatal("atualizações perdidas")
		}
	})

	t.Run("cancelamento durante bloqueio", func(t *testing.T) {
		a := create(t, 0)
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal("falha ao iniciar bloqueio de teste")
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `SELECT id FROM financeiro.accounts WHERE id::text=$1 FOR UPDATE`, a.ID); err != nil {
			t.Fatal("falha ao bloquear fixture")
		}
		short, stop := context.WithTimeout(ctx, 100*time.Millisecond)
		defer stop()
		if _, err := other.Register(short, a.ID, "income", "Fixture", 1, "2026-01-01"); err == nil {
			t.Fatal("timeout não interrompeu espera")
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal("falha ao liberar bloqueio")
		}
		state := get(t, a.ID)
		if len(state.Entries) != 0 || state.Account.BalanceCents != 0 {
			t.Fatal("cancelamento gravou alteração")
		}
	})

	t.Run("conta inexistente e validacao", func(t *testing.T) {
		for _, id := range []string{"ausente", "0", "-1", "01", "9223372036854775808"} {
			if _, err := s.Get(ctx, id); !errors.Is(err, contas.ErrNotFound) {
				t.Fatal("ID inválido não mapeado para conta inexistente")
			}
			if _, err := s.Register(ctx, id, "income", "Fixture", 1, "2026-01-01"); !errors.Is(err, contas.ErrNotFound) {
				t.Fatal("registro em conta inexistente")
			}
		}
		a := create(t, 0)
		if _, err := s.Register(ctx, a.ID, "income", "Fixture", 0, "2026-01-01"); !errors.Is(err, contas.ErrInvalid) {
			t.Fatal("valor zero aceito")
		}
		if _, err := s.Register(ctx, a.ID, "expense", "Fixture", 1, "2025-12-31"); !errors.Is(err, contas.ErrInvalid) {
			t.Fatal("data anterior aceita")
		}
		if len(get(t, a.ID).Entries) != 0 {
			t.Fatal("validação gravou lançamento")
		}
	})
}
