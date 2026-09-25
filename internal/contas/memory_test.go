package contas

import (
	"context"
	"errors"
	"testing"
)

func TestCanceledOperations(t *testing.T) {
	s := NewService(NewMemoryRepository())
	a := newAccount(t, s, 0)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.Create(ctx, "Fixture", 0, "2026-01-01"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, a.ID); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.Register(ctx, a.ID, "income", "Fixture", 1, "2026-01-01"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	list, err := s.List(t.Context())
	if err != nil || len(list) != 1 || list[0].BalanceCents != 0 {
		t.Fatal("operação cancelada alterou estado")
	}
}

func TestUpdatePreservesHistory(t *testing.T) {
	repo := NewMemoryRepository()
	s := NewService(repo)
	a := newAccount(t, s, 0)
	if _, err := s.Register(t.Context(), a.ID, "income", "Fixture", 1, "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	err := repo.Update(t.Context(), a.ID, func(state *State) error {
		state.Entries[0].AmountCents = 99
		state.Entries = append(state.Entries, Entry{})
		return nil
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatal("alteração do histórico aceita")
	}
	state, err := s.Get(t.Context(), a.ID)
	if err != nil || len(state.Entries) != 1 || state.Entries[0].AmountCents != 1 || state.Account.BalanceCents != 1 {
		t.Fatal("erro alterou histórico")
	}
	err = repo.Update(t.Context(), a.ID, func(state *State) error { state.Account.BalanceCents = 999; return ErrInvalid })
	if !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	state, err = s.Get(t.Context(), a.ID)
	if err != nil || state.Account.BalanceCents != 1 {
		t.Fatal("callback com erro gravou saldo")
	}
}
