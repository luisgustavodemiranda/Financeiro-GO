// Package contas reúne contas e seus lançamentos realizados.
package contas

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

var (
	ErrInvalid  = errors.New("dados inválidos")
	ErrNotFound = errors.New("conta não encontrada")
	ErrOverflow = errors.New("saldo excede os limites de int64")
)

type Account struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	InitialBalanceCents int64  `json:"initial_balance_cents"`
	InitialBalanceDate  string `json:"initial_balance_date"`
	BalanceCents        int64  `json:"balance_cents"`
}

type Entry struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
	Date        string `json:"date"`
}

// State é o conjunto consistente de conta e lançamentos.
type State struct {
	Account Account
	Entries []Entry
}

type Repository interface {
	Create(context.Context, Account) (Account, error)
	List(context.Context) ([]Account, error)
	Get(context.Context, string) (State, error)
	// Update executa change de forma exclusiva e só grava se não houver erro.
	// A implementação não deve compartilhar slices internos com o chamador.
	// change só pode acrescentar um lançamento e alterar BalanceCents.
	Update(ctx context.Context, id string, change func(*State) error) error
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Protege o contrato de atualização: histórico e cadastro são imutáveis nesta fase.
func validateAppend(before, after State) error {
	a := after.Account
	a.BalanceCents = before.Account.BalanceCents
	if a != before.Account || len(after.Entries) != len(before.Entries)+1 ||
		!slices.Equal(before.Entries, after.Entries[:len(before.Entries)]) {
		return fmt.Errorf("%w: atualização deve acrescentar somente um lançamento", ErrInvalid)
	}
	return nil
}

func validDate(value string) bool {
	date, err := time.Parse("2006-01-02", value)
	return err == nil && date.Year() >= 1
}

func (s *Service) Create(ctx context.Context, name string, initial int64, date string) (Account, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsRune(name, 0) || !validDate(date) {
		return Account{}, fmt.Errorf("%w: name e initial_balance_date (AAAA-MM-DD) são obrigatórios", ErrInvalid)
	}
	return s.repo.Create(ctx, Account{Name: name, InitialBalanceCents: initial, InitialBalanceDate: date, BalanceCents: initial})
}

func (s *Service) List(ctx context.Context) ([]Account, error)       { return s.repo.List(ctx) }
func (s *Service) Get(ctx context.Context, id string) (State, error) { return s.repo.Get(ctx, id) }

func (s *Service) Register(ctx context.Context, id, kind, description string, amount int64, date string) (Entry, error) {
	description = strings.TrimSpace(description)
	if (kind != "income" && kind != "expense") || description == "" || strings.ContainsRune(description, 0) || amount <= 0 || !validDate(date) {
		return Entry{}, fmt.Errorf("%w: kind deve ser income ou expense, description e date são obrigatórios e amount_cents deve ser positivo", ErrInvalid)
	}
	entry := Entry{AccountID: id, Kind: kind, Description: description, AmountCents: amount, Date: date}
	err := s.repo.Update(ctx, id, func(state *State) error {
		// O saldo inicial representa a abertura do dia de referência.
		if date < state.Account.InitialBalanceDate {
			return fmt.Errorf("%w: date não pode anteceder initial_balance_date", ErrInvalid)
		}
		balance := state.Account.BalanceCents
		if kind == "income" {
			if balance > math.MaxInt64-amount {
				return ErrOverflow
			}
			balance += amount
		} else {
			if balance < math.MinInt64+amount {
				return ErrOverflow
			}
			balance -= amount
		}
		entry.ID = fmt.Sprintf("%s-%d", id, len(state.Entries)+1)
		state.Entries = append(state.Entries, entry)
		state.Account.BalanceCents = balance
		return nil
	})
	if err != nil {
		return Entry{}, err
	}
	return entry, nil
}
