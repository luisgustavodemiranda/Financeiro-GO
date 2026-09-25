package contas

import (
	"errors"
	"math"
	"sync"
	"testing"
)

func newAccount(t *testing.T, s *Service, initial int64) Account {
	t.Helper()
	a, err := s.Create(t.Context(), " Conta fictícia ", initial, "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAccountFlow(t *testing.T) {
	s := NewService(NewMemoryRepository())
	list, err := s.List(t.Context())
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("lista inicial: %v, %v", list, err)
	}
	a := newAccount(t, s, 1000)
	b := newAccount(t, s, 0)
	state, _ := s.Get(t.Context(), a.ID)
	if a.Name != "Conta fictícia" || len(state.Entries) != 0 || a.BalanceCents != 1000 {
		t.Fatal("saldo inicial não deve gerar receita")
	}
	for _, entry := range []struct {
		kind   string
		amount int64
	}{{"income", 500}, {"expense", 1800}} {
		if _, err := s.Register(t.Context(), a.ID, entry.kind, "Exemplo", entry.amount, "2026-01-01"); err != nil {
			t.Fatal(err)
		}
	}
	state, _ = s.Get(t.Context(), a.ID)
	if state.Account.BalanceCents != -300 || state.Account.InitialBalanceCents != 1000 || len(state.Entries) != 2 {
		t.Fatalf("estado: %+v", state)
	}
	if state.Entries[0].ID == state.Entries[1].ID || state.Entries[0].AccountID != a.ID {
		t.Fatal("identificação dos lançamentos")
	}
	state.Entries[0].AmountCents = 99
	state, _ = s.Get(t.Context(), a.ID)
	if state.Entries[0].AmountCents != 500 {
		t.Fatal("slice interno exposto")
	}
	other, _ := s.Get(t.Context(), b.ID)
	if other.Account.BalanceCents != 0 || len(other.Entries) != 0 {
		t.Fatal("contas não isoladas")
	}
	list, _ = s.List(t.Context())
	if len(list) != 2 || list[0].BalanceCents != -300 {
		t.Fatal("listagem inconsistente")
	}
}

func TestValidation(t *testing.T) {
	s := NewService(NewMemoryRepository())
	for _, input := range []struct{ name, date string }{{" ", "2026-01-01"}, {"Exemplo", ""}, {"Exemplo", "2026-02-30"}, {"Exemplo", "0000-01-01"}, {"Exemplo\x00", "2026-01-01"}} {
		if _, err := s.Create(t.Context(), input.name, 0, input.date); !errors.Is(err, ErrInvalid) {
			t.Fatalf("validação: %v", err)
		}
	}
	a := newAccount(t, s, 0)
	for _, tc := range []struct {
		kind, description, date string
		amount                  int64
	}{
		{"transfer", "Exemplo", "2026-01-01", 1}, {"income", " ", "2026-01-01", 1},
		{"income", "Exemplo", "2026-01-01", 0}, {"expense", "Exemplo", "2026-01-01", -1},
		{"income", "Exemplo", "2025-12-31", 1}, {"income", "Exemplo", "inválida", 1},
		{"income", "Exemplo\x00", "2026-01-01", 1},
	} {
		if _, err := s.Register(t.Context(), a.ID, tc.kind, tc.description, tc.amount, tc.date); !errors.Is(err, ErrInvalid) {
			t.Fatalf("validação: %v", err)
		}
	}
	if _, err := s.Register(t.Context(), "ausente", "income", "Exemplo", 1, "2026-01-01"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := s.Get(t.Context(), "ausente"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	state, _ := s.Get(t.Context(), a.ID)
	if state.Account.BalanceCents != 0 || len(state.Entries) != 0 {
		t.Fatal("validação alterou o estado")
	}
}

func TestBalanceLimits(t *testing.T) {
	for _, tc := range []struct {
		name, kind            string
		initial, amount, want int64
		overflow              bool
	}{
		{"max", "income", math.MaxInt64 - 1, 1, math.MaxInt64, false},
		{"min", "expense", math.MinInt64 + 1, 1, math.MinInt64, false},
		{"overflow", "income", math.MaxInt64, 1, math.MaxInt64, true},
		{"underflow", "expense", math.MinInt64, 1, math.MinInt64, true},
		{"large income", "income", math.MinInt64, math.MaxInt64, -1, false},
		{"large expense", "expense", math.MaxInt64, math.MaxInt64, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewService(NewMemoryRepository())
			a := newAccount(t, s, tc.initial)
			_, err := s.Register(t.Context(), a.ID, tc.kind, "Limite fictício", tc.amount, "2026-01-02")
			if tc.overflow {
				if !errors.Is(err, ErrOverflow) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			state, _ := s.Get(t.Context(), a.ID)
			count := 1
			if tc.overflow {
				count = 0
			}
			if state.Account.BalanceCents != tc.want || len(state.Entries) != count {
				t.Fatalf("estado: %+v", state)
			}
		})
	}
}

func TestConcurrentEntries(t *testing.T) {
	s := NewService(NewMemoryRepository())
	a := newAccount(t, s, 0)
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			if _, err := s.Register(t.Context(), a.ID, "income", "Exemplo", 1, "2026-01-01"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	state, _ := s.Get(t.Context(), a.ID)
	if state.Account.BalanceCents != 100 || len(state.Entries) != 100 {
		t.Fatal("atualizações perdidas")
	}
}
