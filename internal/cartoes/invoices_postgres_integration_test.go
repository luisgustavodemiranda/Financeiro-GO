package cartoes_test

import (
	"context"
	"encoding/json"
	"financeirogo/internal/cartoes"
	"financeirogo/internal/contas"
	"financeirogo/internal/database"
	"financeirogo/internal/server"
	"financeirogo/migrations"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPostgresInvoicesIntegration(t *testing.T) {
	raw := os.Getenv("FINANCEIRO_TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("configure financeiro_go_test com migration 0003 previamente aplicada")
	}
	cfg, err := database.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "financeiro_go_test" {
		t.Fatal("integração exige banco exclusivo de testes")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Check(ctx, pool); err != nil {
		t.Fatal(err)
	}
	repo := cartoes.NewPostgresRepository(pool, 10*time.Second)
	cleanup := func(t *testing.T, card string) {
		t.Helper()
		id, err := strconv.ParseInt(card, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Error("limpeza das fixtures falhou")
				return
			}
			defer tx.Rollback(ctx)
			for _, sql := range []string{`DELETE FROM financeiro.purchases WHERE invoice_id IN (SELECT id FROM financeiro.invoices WHERE card_id=$1)`, `DELETE FROM financeiro.invoices WHERE card_id=$1`, `DELETE FROM financeiro.cards WHERE id=$1`} {
				if _, err := tx.Exec(ctx, sql, id); err != nil {
					t.Error("limpeza das próprias fixtures falhou")
					return
				}
			}
			if err := tx.Commit(ctx); err != nil {
				t.Error("commit da limpeza falhou")
			}
		})
	}
	t.Run("contrato compartilhado", func(t *testing.T) { runInvoiceBehavior(t, repo, cleanup) })
	t.Run("HTTP e persistencia em outro pool", func(t *testing.T) {
		s := cartoes.NewService(repo)
		card, err := s.Create(ctx, "Cartao ficticio HTTP faturas")
		if err != nil {
			t.Fatal(err)
		}
		cleanup(t, card.ID)
		h := server.NewHandlerWithRepositories(contas.NewPostgresRepository(pool, 5*time.Second), repo, "postgresql")
		request := func(path, body string, code int) cartoes.Invoice {
			t.Helper()
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(body)))
			if w.Code != code {
				t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
			}
			var result cartoes.Invoice
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			return result
		}
		base := "/api/v1/cards/" + card.ID + "/invoices"
		i := request(base, `{"start_date":"2026-01-01","closing_date":"2026-01-31","due_date":"2026-02-10"}`, 201)
		request(base+"/"+i.ID+"/purchases", `{"description":"Compra ficticia","amount_cents":1234,"date":"2026-01-10"}`, 201)
		request(base+"/"+i.ID+"/close", `{}`, 200)
		other, err := database.Open(ctx, raw)
		if err != nil {
			t.Fatal(err)
		}
		defer other.Close()
		got, err := cartoes.NewService(cartoes.NewPostgresRepository(other, 5*time.Second)).GetInvoice(ctx, card.ID, i.ID)
		if err != nil || got.Status != "closed" || got.TotalCents != 1234 || len(got.Purchases) != 1 {
			t.Fatal("persistência", got, err)
		}
	})
}
