// Package cartoes reúne cartões, compras e faturas sem movimentar contas bancárias.
package cartoes

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrInvalid = errors.New("name deve conter de 1 a 100 caracteres válidos")

type Card struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Repository interface {
	Create(context.Context, Card) (Card, error)
	List(context.Context) ([]Card, error)
	CreateInvoice(context.Context, Invoice) (Invoice, error)
	ListInvoices(context.Context, string) ([]Invoice, error)
	GetInvoice(context.Context, string, string) (Invoice, error)
	// UpdateInvoice serializa compra e fechamento e só grava se change tiver sucesso.
	UpdateInvoice(context.Context, string, string, func(*Invoice) error) (Invoice, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, name string) (Card, error) {
	name = strings.TrimSpace(name)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 100 || strings.ContainsRune(name, 0) {
		return Card{}, ErrInvalid
	}
	return s.repo.Create(ctx, Card{Name: name})
}

func (s *Service) List(ctx context.Context) ([]Card, error) { return s.repo.List(ctx) }
