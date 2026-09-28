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

func TestPostgresPaymentsIntegration(t *testing.T) {
	raw := os.Getenv("FINANCEIRO_TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("configure financeiro_go_test com migration 0004 previamente aplicada")
	}
	cfg, err := database.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "financeiro_go_test" {
		t.Fatal("integração exige banco exclusivo de testes")
	}
	pool, err := database.Open(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Check(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	a, c := contas.NewPostgresRepository(pool, 10*time.Second), cartoes.NewPostgresRepository(pool, 10*time.Second)
	cleanup := func(t *testing.T, account, card string) {
		t.Helper()
		accountID, _ := strconv.ParseInt(account, 10, 64)
		cardID, _ := strconv.ParseInt(card, 10, 64)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Error("limpeza falhou")
				return
			}
			defer tx.Rollback(ctx)
			// Apenas os IDs das fixtures desta chamada. Desfaz a referência circular
			// entre pagamento e lançamento antes de remover seus registros.
			for _, sql := range []string{
				`UPDATE financeiro.invoices SET status='closed',paid_account_id=NULL,paid_date=NULL,paid_entry_id=NULL WHERE card_id=$1`,
				`DELETE FROM financeiro.entries WHERE invoice_id IN (SELECT id FROM financeiro.invoices WHERE card_id=$1)`,
				`DELETE FROM financeiro.purchases WHERE invoice_id IN (SELECT id FROM financeiro.invoices WHERE card_id=$1)`,
				`DELETE FROM financeiro.invoices WHERE card_id=$1`,
				`DELETE FROM financeiro.cards WHERE id=$1`,
			} {
				if _, err := tx.Exec(ctx, sql, cardID); err != nil {
					t.Error("limpeza das fixtures de cartão falhou")
					return
				}
			}
			if _, err := tx.Exec(ctx, `DELETE FROM financeiro.entries WHERE account_id=$1`, accountID); err != nil {
				t.Error("limpeza dos lançamentos falhou")
				return
			}
			if _, err := tx.Exec(ctx, `DELETE FROM financeiro.accounts WHERE id=$1`, accountID); err != nil {
				t.Error("limpeza da conta falhou")
				return
			}
			if err := tx.Commit(ctx); err != nil {
				t.Error("commit da limpeza falhou")
			}
		})
	}
	t.Run("contrato compartilhado", func(t *testing.T) { runPaymentBehavior(t, a, c, c, cleanup) })
	t.Run("rollback apos escritas e HTTP", func(t *testing.T) {
		as, cs := contas.NewService(a), cartoes.NewService(c)
		account, err := as.Create(t.Context(), "Ficticia integracao pagamento", 100, "2026-01-01")
		if err != nil {
			t.Fatal(err)
		}
		card, err := cs.Create(t.Context(), "Ficticio integracao pagamento")
		if err != nil {
			t.Fatal(err)
		}
		cleanup(t, account.ID, card.ID)
		i, err := cs.CreateInvoice(t.Context(), card.ID, "2026-01-01", "2026-01-31", "2026-02-10")
		if err != nil {
			t.Fatal(err)
		}
		i, err = cs.RegisterPurchase(t.Context(), card.ID, i.ID, "Ficticia", 10, "2026-01-10")
		if err != nil {
			t.Fatal(err)
		}
		i, err = cs.CloseInvoice(t.Context(), card.ID, i.ID)
		if err != nil {
			t.Fatal(err)
		}
		// Força falha na constraint da última escrita: INSERT do lançamento e
		// UPDATE da conta já executaram, mas devem ser revertidos pela transação.
		_, err = c.PayInvoice(t.Context(), card.ID, i.ID, account.ID, func(i *cartoes.Invoice, a *contas.Account) error {
			a.BalanceCents -= 10
			i.Status = "paid"
			i.Payment = &cartoes.Payment{AccountID: a.ID, Date: "2026-01-20", AmountCents: 10}
			return nil
		})
		if err == nil {
			t.Fatal("constraint não rejeitou data anterior ao fechamento")
		}
		state, err := as.Get(t.Context(), account.ID)
		if err != nil || state.Account.BalanceCents != 100 || len(state.Entries) != 0 {
			t.Fatal("rollback da conta", state, err)
		}
		got, err := cs.GetInvoice(t.Context(), card.ID, i.ID)
		if err != nil || got.Status != "closed" || got.Payment != nil {
			t.Fatal("rollback da fatura", got, err)
		}
		h := server.NewHandlerWithRepositories(a, c, c, "postgresql")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/cards/"+card.ID+"/invoices/"+i.ID+"/payment", strings.NewReader(`{"account_id":"`+account.ID+`","date":"2026-02-10"}`)))
		var paid cartoes.Invoice
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &paid) != nil || paid.Payment == nil {
			t.Fatal("HTTP", w.Code, w.Body.String())
		}
		other, err := database.Open(t.Context(), raw)
		if err != nil {
			t.Fatal(err)
		}
		defer other.Close()
		got, err = cartoes.NewService(cartoes.NewPostgresRepository(other, 5*time.Second)).GetInvoice(t.Context(), card.ID, i.ID)
		if err != nil || got.Status != "paid" || got.Payment == nil || *got.Payment != *paid.Payment {
			t.Fatal("persistência", got, err)
		}
	})
}
