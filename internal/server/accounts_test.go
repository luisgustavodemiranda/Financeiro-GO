package server

import (
	"encoding/json"
	"financeirogo/internal/contas"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func request(t *testing.T, h http.Handler, method, path, body string, status int) []byte {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	if w.Code != status {
		t.Fatalf("%s %s: status %d, esperado %d: %s", method, path, w.Code, status, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Fatal("Content-Type")
	}
	if !json.Valid(w.Body.Bytes()) {
		t.Fatal("JSON inválido na resposta")
	}
	return w.Body.Bytes()
}

func TestAccountEndpoints(t *testing.T) {
	h := NewHandler()
	if string(request(t, h, "GET", "/api/v1/accounts", "", 200)) != "[]\n" {
		t.Fatal("lista vazia")
	}
	request(t, h, "POST", "/api/v1/accounts", `{"name":"Conta fictícia","initial_balance_cents":1000,"initial_balance_date":"2026-01-01"}`, 201)
	if string(request(t, h, "GET", "/api/v1/accounts/1/entries", "", 200)) != "[]\n" {
		t.Fatal("saldo inicial gerou receita")
	}
	request(t, h, "POST", "/api/v1/accounts/1/entries", `{"kind":"income","description":"Receita fictícia","amount_cents":500,"date":"2026-01-01"}`, 201)
	request(t, h, "POST", "/api/v1/accounts/1/entries", `{"kind":"expense","description":"Despesa fictícia","amount_cents":1800,"date":"2026-01-02"}`, 201)
	var a contas.Account
	if err := json.Unmarshal(request(t, h, "GET", "/api/v1/accounts/1/balance", "", 200), &a); err != nil {
		t.Fatal(err)
	}
	if a.BalanceCents != -300 || a.InitialBalanceCents != 1000 {
		t.Fatalf("saldo: %+v", a)
	}
	var entries []contas.Entry
	if err := json.Unmarshal(request(t, h, "GET", "/api/v1/accounts/1/entries", "", 200), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].AmountCents != 500 || entries[1].Kind != "expense" {
		t.Fatal(entries)
	}
	var accounts []contas.Account
	if err := json.Unmarshal(request(t, h, "GET", "/api/v1/accounts", "", 200), &accounts); err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].BalanceCents != -300 {
		t.Fatal(accounts)
	}
	request(t, NewHandler(), "GET", "/api/v1/accounts/1/balance", "", 404)
}

func TestInvalidAccountJSON(t *testing.T) {
	h := NewHandler()
	for _, body := range []string{
		`{`, `null`, `{}`, `{"name":"Exemplo","initial_balance_date":"2026-01-01"}`,
		`{"name":"Exemplo","initial_balance_cents":null,"initial_balance_date":"2026-01-01"}`,
		`{"name":"Exemplo","initial_balance_cents":1.5,"initial_balance_date":"2026-01-01"}`,
		`{"name":"Exemplo","initial_balance_cents":"100","initial_balance_date":"2026-01-01"}`,
		`{"name":"Exemplo","initial_balance_cents":9223372036854775808,"initial_balance_date":"2026-01-01"}`,
		`{"name":"Exemplo","initial_balance_cents":0,"initial_balance_date":"2026-01-01","extra":true}`,
		`{"name":"Exemplo","initial_balance_cents":0,"initial_balance_date":"2026-01-01"} {}`,
	} {
		request(t, h, "POST", "/api/v1/accounts", body, 400)
	}
	if string(request(t, h, "GET", "/api/v1/accounts", "", 200)) != "[]\n" {
		t.Fatal("entrada inválida criou conta")
	}
}

func TestEntryErrorsAndOverflow(t *testing.T) {
	h := NewHandler()
	request(t, h, "POST", "/api/v1/accounts", `{"name":"Limite fictício","initial_balance_cents":9223372036854775807,"initial_balance_date":"2026-01-01"}`, 201)
	valid := `{"kind":"income","description":"Exemplo","amount_cents":1,"date":"2026-01-01"}`
	request(t, h, "POST", "/api/v1/accounts/1/entries", valid, 409)
	request(t, h, "POST", "/api/v1/accounts/missing/entries", valid, 404)
	for _, path := range []string{"entries", "balance"} {
		request(t, h, "GET", "/api/v1/accounts/missing/"+path, "", 404)
	}
	for _, body := range []string{`{}`, `null`, strings.Replace(valid, `"amount_cents":1`, `"amount_cents":0`, 1), strings.Replace(valid, `"amount_cents":1`, `"amount_cents":-1`, 1), strings.Replace(valid, `"amount_cents":1`, `"amount_cents":1.2`, 1), strings.Replace(valid, "income", "transfer", 1), strings.Replace(valid, "Exemplo", " ", 1), strings.Replace(valid, "2026-01-01", "2025-01-01", 1)} {
		request(t, h, "POST", "/api/v1/accounts/1/entries", body, 400)
	}
	var a contas.Account
	if err := json.Unmarshal(request(t, h, "GET", "/api/v1/accounts/1/balance", "", 200), &a); err != nil {
		t.Fatal(err)
	}
	if a.BalanceCents != 9223372036854775807 {
		t.Fatal("perda de precisão ou alteração indevida")
	}
	if string(request(t, h, "GET", "/api/v1/accounts/1/entries", "", 200)) != "[]\n" {
		t.Fatal("falha gravou lançamento")
	}
}
