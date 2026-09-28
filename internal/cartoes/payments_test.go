package cartoes_test

import (
	"context"
	"errors"
	"financeirogo/internal/cartoes"
	"financeirogo/internal/contas"
	"math"
	"reflect"
	"testing"
)

func TestMemoryPayments(t *testing.T) {
	a, c := contas.NewMemoryRepository(), cartoes.NewMemoryRepository()
	runPaymentBehavior(t, a, c, cartoes.NewMemoryPaymentRepository(a, c), func(*testing.T, string, string) {})
}

func runPaymentBehavior(t *testing.T, accounts contas.Repository, cards cartoes.Repository, payments cartoes.PaymentRepository, cleanup func(*testing.T, string, string)) {
	as, cs, ps := contas.NewService(accounts), cartoes.NewService(cards), cartoes.NewPaymentService(payments)
	setup := func(t *testing.T, balance, amount int64, closed bool) (contas.Account, cartoes.Invoice) {
		t.Helper()
		a, err := as.Create(t.Context(), "Conta ficticia pagamento", balance, "2026-01-01")
		if err != nil {
			t.Fatal(err)
		}
		c, err := cs.Create(t.Context(), "Cartao ficticio pagamento")
		if err != nil {
			t.Fatal(err)
		}
		cleanup(t, a.ID, c.ID)
		i, err := cs.CreateInvoice(t.Context(), c.ID, "2026-01-01", "2026-01-31", "2026-02-10")
		if err != nil {
			t.Fatal(err)
		}
		if amount > 0 {
			i, err = cs.RegisterPurchase(t.Context(), c.ID, i.ID, "Compra ficticia", amount, "2026-01-10")
			if err != nil {
				t.Fatal(err)
			}
		}
		if closed {
			i, err = cs.CloseInvoice(t.Context(), c.ID, i.ID)
			if err != nil {
				t.Fatal(err)
			}
		}
		return a, i
	}
	t.Run("pagamento e repeticao", func(t *testing.T) {
		a, i := setup(t, 1000, 1234, true)
		paid, err := ps.Pay(t.Context(), i.CardID, i.ID, a.ID, "2026-02-10")
		if err != nil {
			t.Fatal(err)
		}
		if paid.Status != "paid" || paid.TotalCents != 1234 || paid.Payment == nil || paid.Payment.AmountCents != 1234 {
			t.Fatal(paid)
		}
		state, err := as.Get(t.Context(), a.ID)
		if err != nil {
			t.Fatal(err)
		}
		if state.Account.BalanceCents != -234 || len(state.Entries) != 1 || state.Entries[0].Kind != "invoice_payment" || state.Entries[0].InvoiceID != i.ID || state.Entries[0].ID != paid.Payment.EntryID {
			t.Fatal(state)
		}
		again, err := ps.Pay(t.Context(), i.CardID, i.ID, a.ID, "2026-02-10")
		if err != nil || !reflect.DeepEqual(paid, again) {
			t.Fatal("repetição", err)
		}
		if _, err := ps.Pay(t.Context(), i.CardID, i.ID, a.ID, "2026-02-11"); !errors.Is(err, cartoes.ErrPaymentConflict) {
			t.Fatal(err)
		}
		other, err := as.Create(t.Context(), "Outra ficticia", 100, "2026-01-01")
		if err != nil {
			t.Fatal(err)
		}
		cleanup(t, other.ID, "")
		if _, err := ps.Pay(t.Context(), i.CardID, i.ID, other.ID, "2026-02-10"); !errors.Is(err, cartoes.ErrPaymentConflict) {
			t.Fatal(err)
		}
		after, err := as.Get(t.Context(), a.ID)
		if err != nil || !reflect.DeepEqual(state, after) {
			t.Fatal("duplicou débito", err)
		}
		closed, err := cs.CloseInvoice(t.Context(), i.CardID, i.ID)
		if err != nil || closed.Status != "paid" {
			t.Fatal("rebaixou fatura paga", err)
		}
		if _, err := cs.RegisterPurchase(t.Context(), i.CardID, i.ID, "Ficticia", 1, "2026-01-15"); !errors.Is(err, cartoes.ErrConflict) {
			t.Fatal(err)
		}
		paid.Payment.Date = "alterado"
		got, err := cs.GetInvoice(t.Context(), i.CardID, i.ID)
		if err != nil || got.Payment.Date != "2026-02-10" {
			t.Fatal("cópia", err)
		}
	})
	t.Run("falhas preservam ambos", func(t *testing.T) {
		for _, tc := range []struct {
			name            string
			balance, amount int64
			closed          bool
			date            string
			expected        error
		}{
			{"aberta", 100, 10, false, "2026-02-10", cartoes.ErrPaymentConflict},
			{"vazia", 100, 0, true, "2026-02-10", cartoes.ErrPaymentConflict},
			{"overflow", math.MinInt64, 1, true, "2026-02-10", contas.ErrOverflow},
			{"antes fechamento", 100, 10, true, "2026-01-30", cartoes.ErrInvoiceInvalid},
			{"data inválida", 100, 10, true, "2026-02-30", cartoes.ErrInvoiceInvalid},
		} {
			t.Run(tc.name, func(t *testing.T) {
				a, i := setup(t, tc.balance, tc.amount, tc.closed)
				before, err := as.Get(t.Context(), a.ID)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := ps.Pay(t.Context(), i.CardID, i.ID, a.ID, tc.date); !errors.Is(err, tc.expected) {
					t.Fatal(err)
				}
				after, err := as.Get(t.Context(), a.ID)
				if err != nil || !reflect.DeepEqual(before, after) {
					t.Fatal("conta alterada", err)
				}
				got, err := cs.GetInvoice(t.Context(), i.CardID, i.ID)
				if err != nil || !reflect.DeepEqual(i, got) {
					t.Fatal("fatura alterada", err)
				}
			})
		}
	})
	t.Run("limite exato e datas", func(t *testing.T) {
		a, i := setup(t, math.MinInt64+1, 1, true)
		if _, err := ps.Pay(t.Context(), i.CardID, i.ID, a.ID, "2026-01-31"); err != nil {
			t.Fatal(err)
		}
		state, err := as.Get(t.Context(), a.ID)
		if err != nil || state.Account.BalanceCents != math.MinInt64 {
			t.Fatal(state, err)
		}
		late, err := as.Create(t.Context(), "Ficticia posterior", 100, "2026-03-01")
		if err != nil {
			t.Fatal(err)
		}
		cleanup(t, late.ID, "")
		_, fresh := setup(t, 100, 10, true)
		if _, err := ps.Pay(t.Context(), fresh.CardID, fresh.ID, late.ID, "2026-02-10"); !errors.Is(err, cartoes.ErrInvoiceInvalid) {
			t.Fatal(err)
		}
	})
	t.Run("ausencia e cancelamento", func(t *testing.T) {
		a, i := setup(t, 100, 10, true)
		if _, err := ps.Pay(t.Context(), i.CardID, i.ID, "999999999", "2026-02-10"); !errors.Is(err, contas.ErrNotFound) {
			t.Fatal(err)
		}
		if _, err := ps.Pay(t.Context(), i.CardID, "999999999", a.ID, "2026-02-10"); !errors.Is(err, cartoes.ErrNotFound) {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if _, err := ps.Pay(ctx, i.CardID, i.ID, a.ID, "2026-02-10"); err == nil {
			t.Fatal("cancelamento ignorado")
		}
		state, err := as.Get(t.Context(), a.ID)
		if err != nil || state.Account.BalanceCents != 100 || len(state.Entries) != 0 {
			t.Fatal(state, err)
		}
	})
	t.Run("duas faturas na mesma conta", func(t *testing.T) {
		a, first := setup(t, 1000, 100, true)
		_, second := setup(t, 1000, 200, true)
		results := make(chan error, 2)
		for _, i := range []cartoes.Invoice{first, second} {
			go func() { _, err := ps.Pay(t.Context(), i.CardID, i.ID, a.ID, "2026-02-10"); results <- err }()
		}
		for range 2 {
			if err := <-results; err != nil {
				t.Fatal(err)
			}
		}
		state, err := as.Get(t.Context(), a.ID)
		if err != nil || state.Account.BalanceCents != 700 || len(state.Entries) != 2 {
			t.Fatal(state, err)
		}
	})
	t.Run("concorrencia e lancamentos normais", func(t *testing.T) {
		a, i := setup(t, 1000, 100, true)
		results := make(chan error, 24)
		for range 12 {
			go func() { _, err := ps.Pay(t.Context(), i.CardID, i.ID, a.ID, "2026-02-10"); results <- err }()
		}
		for range 12 {
			go func() {
				_, err := as.Register(t.Context(), a.ID, "income", "Receita ficticia", 1, "2026-02-10")
				results <- err
			}()
		}
		for range 24 {
			if err := <-results; err != nil {
				t.Error(err)
			}
		}
		state, err := as.Get(t.Context(), a.ID)
		if err != nil || state.Account.BalanceCents != 912 || len(state.Entries) != 13 {
			t.Fatal("saldo/lançamentos", state, err)
		}
		payments := 0
		ids := map[string]bool{}
		for _, e := range state.Entries {
			if ids[e.ID] {
				t.Fatal("ID duplicado")
			}
			ids[e.ID] = true
			if e.Kind == "invoice_payment" {
				payments++
			}
			if e.Kind == "expense" {
				t.Fatal("duplicou despesa")
			}
		}
		if payments != 1 {
			t.Fatal("pagamentos", payments)
		}
	})
}

func TestMemoryPaymentCancellationBeforeCommit(t *testing.T) {
	a, c := contas.NewMemoryRepository(), cartoes.NewMemoryRepository()
	as, cs := contas.NewService(a), cartoes.NewService(c)
	account, _ := as.Create(t.Context(), "Ficticia", 100, "2026-01-01")
	card, _ := cs.Create(t.Context(), "Ficticio")
	i, _ := cs.CreateInvoice(t.Context(), card.ID, "2026-01-01", "2026-01-31", "2026-02-10")
	i, _ = cs.RegisterPurchase(t.Context(), card.ID, i.ID, "Ficticia", 10, "2026-01-10")
	i, _ = cs.CloseInvoice(t.Context(), card.ID, i.ID)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, err := cartoes.NewMemoryPaymentRepository(a, c).PayInvoice(ctx, card.ID, i.ID, account.ID, func(i *cartoes.Invoice, a *contas.Account) error {
		a.BalanceCents -= 10
		i.Status = "paid"
		i.Payment = &cartoes.Payment{AccountID: a.ID, Date: "2026-02-10", AmountCents: 10}
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	state, err := as.Get(t.Context(), account.ID)
	if err != nil || state.Account.BalanceCents != 100 || len(state.Entries) != 0 {
		t.Fatal(state, err)
	}
	after, err := cs.GetInvoice(t.Context(), card.ID, i.ID)
	if err != nil || !reflect.DeepEqual(i, after) {
		t.Fatal(after, err)
	}
}
