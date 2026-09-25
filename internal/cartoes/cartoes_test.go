package cartoes

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestCreateAndList(t *testing.T) {
	s := NewService(NewMemoryRepository())
	list, err := s.List(t.Context())
	if err != nil || list == nil || len(list) != 0 {
		t.Fatal("lista inicial inválida")
	}
	a, err := s.Create(t.Context(), "  Cartão fictício  ")
	if err != nil || a.ID == "" || a.Name != "Cartão fictício" {
		t.Fatal("cadastro inválido")
	}
	b, err := s.Create(t.Context(), a.Name)
	if err != nil || b.ID == a.ID {
		t.Fatal("nomes iguais devem poder identificar cartões distintos")
	}
	list, err = s.List(t.Context())
	if err != nil || len(list) != 2 || list[0] != a || list[1] != b {
		t.Fatal("ordem incorreta")
	}
	list[0].Name = "Modificação externa"
	list, _ = s.List(t.Context())
	if list[0] != a {
		t.Fatal("estado interno exposto")
	}
}

func TestValidation(t *testing.T) {
	s := NewService(NewMemoryRepository())
	for _, name := range []string{"", " \t\n", strings.Repeat("a", 101), "Exemplo\x00", string([]byte{0xff})} {
		if _, err := s.Create(t.Context(), name); !errors.Is(err, ErrInvalid) {
			t.Fatal("nome inválido aceito")
		}
	}
	if _, err := s.Create(t.Context(), strings.Repeat("á", 100)); err != nil {
		t.Fatal("limite deve contar caracteres, não bytes")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.Create(ctx, "Fictício"); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelamento ignorado")
	}
	if _, err := s.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelamento ignorado")
	}
	list, _ := s.List(t.Context())
	if len(list) != 1 {
		t.Fatal("erro gravou cartão")
	}
}

func TestConcurrentCreate(t *testing.T) {
	s := NewService(NewMemoryRepository())
	var wg sync.WaitGroup
	for range 30 {
		wg.Go(func() {
			if _, err := s.Create(t.Context(), "Fictício"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	list, err := s.List(t.Context())
	if err != nil || len(list) != 30 {
		t.Fatal("cadastros perdidos")
	}
	seen := map[string]bool{}
	for _, c := range list {
		if seen[c.ID] {
			t.Fatal("ID duplicado")
		}
		seen[c.ID] = true
	}
}
