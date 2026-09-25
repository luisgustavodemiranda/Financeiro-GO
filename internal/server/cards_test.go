package server

import (
	"encoding/json"
	"financeirogo/internal/cartoes"
	"testing"
)

func TestCardEndpoints(t *testing.T) {
	h := NewHandler()
	request(t, h, "POST", "/api/v1/accounts", `{"name":"Conta ficticia","initial_balance_cents":100,"initial_balance_date":"2026-01-01"}`, 201)
	before := string(request(t, h, "GET", "/api/v1/accounts/1/balance", "", 200))
	if string(request(t, h, "GET", "/api/v1/cards", "", 200)) != "[]\n" {
		t.Fatal("lista inicial")
	}
	var card cartoes.Card
	if err := json.Unmarshal(request(t, h, "POST", "/api/v1/cards", `{"name":"Cartao ficticio"}`, 201), &card); err != nil {
		t.Fatal(err)
	}
	if card.ID == "" || card.Name != "Cartao ficticio" {
		t.Fatal("contrato incorreto")
	}
	var list []cartoes.Card
	if err := json.Unmarshal(request(t, h, "GET", "/api/v1/cards", "", 200), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != card {
		t.Fatal("cartão não listado")
	}
	if string(request(t, h, "GET", "/api/v1/accounts/1/balance", "", 200)) != before {
		t.Fatal("cadastro de cartão alterou o saldo da conta")
	}
	if string(request(t, NewHandler(), "GET", "/api/v1/cards", "", 200)) != "[]\n" {
		t.Fatal("memória compartilhada entre handlers")
	}
}

func TestInvalidCardEndpoints(t *testing.T) {
	h := NewHandler()
	for _, body := range []string{`null`, `{}`, `{"name":" "}`, `{"name":null}`, `{"name":123}`, `{"name":"Exemplo","number":"ficticio"}`, `{"name":"Exemplo"} {}`, `{`, `{"name":"a\u0000"}`} {
		request(t, h, "POST", "/api/v1/cards", body, 400)
	}
	if string(request(t, h, "GET", "/api/v1/cards", "", 200)) != "[]\n" {
		t.Fatal("erro gravou cartão")
	}
}
