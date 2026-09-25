package server

import (
	"context"
	"errors"
	"financeirogo/internal/cartoes"
	"financeirogo/internal/contas"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingRepository struct {
	contas.Repository
	seen context.Context
}

func (r *failingRepository) List(ctx context.Context) ([]contas.Account, error) {
	r.seen = ctx
	return nil, errors.New("detalhe interno fictício que não deve sair no HTTP")
}

func TestInjectedRepositoryAndSafeError(t *testing.T) {
	repo := &failingRepository{}
	h := NewHandlerWithRepositories(repo, cartoes.NewMemoryRepository(), "postgresql")
	request := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if repo.seen != request.Context() {
		t.Fatal("contexto da requisição não chegou ao repositório")
	}
	if w.Code != 500 || strings.TrimSpace(w.Body.String()) != `{"error":"erro interno"}` {
		t.Fatal("resposta deve omitir erro interno")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/status", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"persistence":"postgresql"`) {
		t.Fatal("modo de persistência incorreto")
	}
}

type failingCardRepository struct {
	cartoes.Repository
	seen context.Context
}

func (r *failingCardRepository) List(ctx context.Context) ([]cartoes.Card, error) {
	r.seen = ctx
	return nil, errors.New("detalhe interno fictício do repositório")
}

func TestCardRepositoryContextAndSafeError(t *testing.T) {
	repo := &failingCardRepository{}
	h := NewHandlerWithRepositories(contas.NewMemoryRepository(), repo, "postgresql")
	req := httptest.NewRequest("GET", "/api/v1/cards", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if repo.seen != req.Context() || w.Code != 500 || strings.TrimSpace(w.Body.String()) != `{"error":"erro interno"}` {
		t.Fatal("contexto ou resposta de erro incorretos")
	}
}
