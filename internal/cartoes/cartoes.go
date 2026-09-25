// Package cartoes mantém o cadastro de cartões, sem movimentações financeiras.
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
