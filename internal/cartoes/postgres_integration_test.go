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
	"strings"
	"testing"
	"time"
)

func TestPostgresCardsIntegration(t *testing.T) {
	raw := os.Getenv("FINANCEIRO_TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("configure financeiro_go_test com migrations previamente aplicadas")
	}
	cfg, err := database.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "financeiro_go_test" {
		t.Fatal("integração exige banco exclusivo de testes")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Check(ctx, pool); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := pool.Exec(cleanup, `DELETE FROM financeiro.cards WHERE id::text=ANY($1::text[])`, ids); err != nil {
			t.Error("limpeza das próprias fixtures falhou")
		}
	}()
	h := server.NewHandlerWithRepositories(contas.NewPostgresRepository(pool, 5*time.Second), cartoes.NewPostgresRepository(pool, 5*time.Second), "postgresql")
	for range 2 {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/cards", strings.NewReader(`{"name":"Cartao ficticio de integracao"}`)))
		var card cartoes.Card
		if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &card) != nil || card.ID == "" {
			t.Fatal("cadastro HTTP não persistiu")
		}
		ids = append(ids, card.ID)
	}
	if ids[0] == ids[1] {
		t.Fatal("IDs repetidos")
	}
	other, err := database.Open(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	list, err := cartoes.NewService(cartoes.NewPostgresRepository(other, 5*time.Second)).List(ctx)
	if err != nil {
		t.Fatal("consulta por nova conexão falhou")
	}
	found := 0
	for _, card := range list {
		if card.ID == ids[0] || card.ID == ids[1] {
			found++
			if card.Name != "Cartao ficticio de integracao" {
				t.Fatal("nome não preservado")
			}
		}
	}
	if found != 2 {
		t.Fatal("cartões não persistiram entre pools")
	}
	// O SQL também precisa rejeitar nome inválido, independentemente do serviço.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("transação de validação falhou")
	}
	_, err = tx.Exec(ctx, `INSERT INTO financeiro.cards(name) VALUES('')`)
	if err == nil {
		t.Error("CHECK não rejeitou nome vazio")
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal("rollback de validação falhou")
	}
}
