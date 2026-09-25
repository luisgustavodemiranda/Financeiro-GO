package cartoes_test

import (
	"context"
	"errors"
	"financeirogo/internal/cartoes"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestMemoryInvoiceBehavior(t *testing.T) {
	runInvoiceBehavior(t, cartoes.NewMemoryRepository(), func(*testing.T, string) {})
}

// A mesma suíte verifica o contrato dos adaptadores, com fixtures sempre fictícias.
func runInvoiceBehavior(t *testing.T, repo cartoes.Repository, cleanup func(*testing.T, string)) {
	s := cartoes.NewService(repo)
	setup := func(t *testing.T) (string, cartoes.Invoice) {
		t.Helper()
		card, err := s.Create(t.Context(), "Cartao ficticio de faturas")
		if err != nil {
			t.Fatal(err)
		}
		cleanup(t, card.ID)
		invoice, err := s.CreateInvoice(t.Context(), card.ID, "2026-01-01", "2026-01-31", "2026-02-10")
		if err != nil {
			t.Fatal(err)
		}
		return card.ID, invoice
	}

	t.Run("datas e periodos", func(t *testing.T) {
		card, first := setup(t)
		for _, dates := range [][3]string{{"2026-02-30", "2026-03-10", "2026-03-20"}, {"0000-01-01", "2026-03-10", "2026-03-20"}, {"2026-02-10", "2026-02-01", "2026-02-20"}, {"2026-02-01", "2026-02-10", "2026-02-10"}, {"2026-02-01", "2026-02-10", "2026-02-09"}, {"", "2026-02-10", "2026-02-20"}} {
			if _, err := s.CreateInvoice(t.Context(), card, dates[0], dates[1], dates[2]); !errors.Is(err, cartoes.ErrInvoiceInvalid) {
				t.Fatalf("datas %v: %v", dates, err)
			}
		}
		for _, dates := range [][2]string{{"2026-01-31", "2026-02-28"}, {"2025-12-01", "2026-01-01"}, {"2026-01-10", "2026-01-20"}, {"2025-12-01", "2026-02-28"}} {
			if _, err := s.CreateInvoice(t.Context(), card, dates[0], dates[1], "2026-03-10"); !errors.Is(err, cartoes.ErrConflict) {
				t.Fatalf("sobreposição: %v", err)
			}
		}
		if _, err := s.CreateInvoice(t.Context(), card, "2026-02-01", "2026-02-28", "2026-03-10"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateInvoice(t.Context(), card, "2026-03-01", "2026-03-01", "2026-03-02"); err != nil {
			t.Fatal("período de um dia", err)
		}
		if first.Status != "open" || first.TotalCents != 0 || first.Purchases == nil {
			t.Fatal(first)
		}
		list, err := s.ListInvoices(t.Context(), card)
		if err != nil || len(list) != 3 {
			t.Fatal("lista", err)
		}
	})
	t.Run("compras limites e fechamento", func(t *testing.T) {
		card, invoice := setup(t)
		for _, date := range []string{"2026-01-01", "2026-01-31"} {
			var err error
			invoice, err = s.RegisterPurchase(t.Context(), card, invoice.ID, " Compra ficticia ", 123, date)
			if err != nil {
				t.Fatal(err)
			}
		}
		if invoice.TotalCents != 246 || len(invoice.Purchases) != 2 || invoice.Purchases[0].Description != "Compra ficticia" || invoice.Purchases[0].ID == invoice.Purchases[1].ID {
			t.Fatal(invoice)
		}
		closed, err := s.CloseInvoice(t.Context(), card, invoice.ID)
		if err != nil || closed.Status != "closed" || closed.TotalCents != 246 {
			t.Fatal(closed, err)
		}
		again, err := s.CloseInvoice(t.Context(), card, invoice.ID)
		if err != nil || !reflect.DeepEqual(again, closed) {
			t.Fatal("fechamento não idempotente", err)
		}
		if _, err := s.RegisterPurchase(t.Context(), card, invoice.ID, "Ficticia", 1, "2026-01-10"); !errors.Is(err, cartoes.ErrConflict) {
			t.Fatal(err)
		}
		_, empty := setup(t)
		if closed, err := s.CloseInvoice(t.Context(), empty.CardID, empty.ID); err != nil || closed.TotalCents != 0 {
			t.Fatal("fechamento vazio", err)
		}
	})
	t.Run("erros preservam estado", func(t *testing.T) {
		card, before := setup(t)
		for _, tc := range []struct {
			description string
			amount      int64
			date        string
		}{
			{"", 1, "2026-01-01"}, {" ", 1, "2026-01-01"}, {"a\x00", 1, "2026-01-01"}, {strings.Repeat("á", 201), 1, "2026-01-01"}, {"Ficticia", 0, "2026-01-01"}, {"Ficticia", -1, "2026-01-01"}, {"Ficticia", 1, "2025-12-31"}, {"Ficticia", 1, "2026-02-01"}, {"Ficticia", 1, "2026-01-32"},
		} {
			if _, err := s.RegisterPurchase(t.Context(), card, before.ID, tc.description, tc.amount, tc.date); !errors.Is(err, cartoes.ErrInvoiceInvalid) {
				t.Fatal(tc, err)
			}
		}
		failure := errors.New("falha fictícia")
		if _, err := repo.UpdateInvoice(t.Context(), card, before.ID, func(i *cartoes.Invoice) error { i.Status = "closed"; return failure }); !errors.Is(err, failure) {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if _, err := s.RegisterPurchase(ctx, card, before.ID, "Ficticia", 1, "2026-01-01"); err == nil {
			t.Fatal("contexto cancelado aceito")
		}
		after, err := s.GetInvoice(t.Context(), card, before.ID)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("erro alterou estado", err)
		}
	})
	t.Run("overflow e copias", func(t *testing.T) {
		card, i := setup(t)
		before, err := s.RegisterPurchase(t.Context(), card, i.ID, "Ficticia", math.MaxInt64, "2026-01-01")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.RegisterPurchase(t.Context(), card, i.ID, "Ficticia", 1, "2026-01-01"); !errors.Is(err, cartoes.ErrOverflow) {
			t.Fatal(err)
		}
		after, err := s.GetInvoice(t.Context(), card, i.ID)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("overflow gravou", err)
		}
		after.Purchases[0].AmountCents = 1
		list, err := s.ListInvoices(t.Context(), card)
		if err != nil || list[0].Purchases[0].AmountCents != math.MaxInt64 {
			t.Fatal("retorno compartilha memória", err)
		}
	})
	t.Run("identidade", func(t *testing.T) {
		card, i := setup(t)
		other, _ := setup(t)
		for _, id := range []string{"inexistente", "0", "-1", "9223372036854775808"} {
			if _, err := s.CreateInvoice(t.Context(), id, "2026-02-01", "2026-02-28", "2026-03-10"); !errors.Is(err, cartoes.ErrNotFound) {
				t.Fatal(err)
			}
			if _, err := s.ListInvoices(t.Context(), id); !errors.Is(err, cartoes.ErrNotFound) {
				t.Fatal(err)
			}
			if _, err := s.GetInvoice(t.Context(), card, id); !errors.Is(err, cartoes.ErrNotFound) {
				t.Fatal(err)
			}
		}
		if _, err := s.RegisterPurchase(t.Context(), other, i.ID, "Ficticia", 1, "2026-01-01"); !errors.Is(err, cartoes.ErrNotFound) {
			t.Fatal(err)
		}
		if _, err := s.CloseInvoice(t.Context(), other, i.ID); !errors.Is(err, cartoes.ErrNotFound) {
			t.Fatal(err)
		}
	})
	t.Run("criacao concorrente", func(t *testing.T) {
		card, _ := setup(t)
		results := make(chan error, 8)
		for range 8 {
			go func() {
				_, err := s.CreateInvoice(t.Context(), card, "2026-02-01", "2026-02-28", "2026-03-10")
				results <- err
			}()
		}
		created := 0
		for range 8 {
			err := <-results
			if err == nil {
				created++
			} else if !errors.Is(err, cartoes.ErrConflict) {
				t.Error(err)
			}
		}
		if created != 1 {
			t.Fatal("faturas sobrepostas", created)
		}
	})
	t.Run("compras e fechamento concorrentes", func(t *testing.T) {
		card, i := setup(t)
		results := make(chan error, 20)
		var wg sync.WaitGroup
		for range 20 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := s.RegisterPurchase(t.Context(), card, i.ID, "Ficticia", 7, "2026-01-01")
				results <- err
			}()
		}
		wg.Wait()
		for range 20 {
			if err := <-results; err != nil {
				t.Fatal(err)
			}
		}
		for range 20 {
			go func() {
				_, err := s.RegisterPurchase(t.Context(), card, i.ID, "Ficticia", 7, "2026-01-01")
				results <- err
			}()
		}
		if _, err := s.CloseInvoice(t.Context(), card, i.ID); err != nil {
			t.Fatal(err)
		}
		accepted := 20
		for range 20 {
			err := <-results
			if err == nil {
				accepted++
			} else if !errors.Is(err, cartoes.ErrConflict) {
				t.Error(err)
			}
		}
		end, err := s.GetInvoice(t.Context(), card, i.ID)
		if err != nil || end.Status != "closed" || end.TotalCents != int64(accepted)*7 || len(end.Purchases) != accepted {
			t.Fatal("estado inconsistente", end, err)
		}
	})
}
