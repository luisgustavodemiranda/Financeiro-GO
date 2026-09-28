package server

import (
	"context"
	"encoding/json"
	"errors"
	"financeirogo/internal/cartoes"
	"financeirogo/internal/contas"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingPaymentRepository struct{ seen context.Context }

func (r *failingPaymentRepository) PayInvoice(ctx context.Context, _, _, _ string, _ func(*cartoes.Invoice, *contas.Account) error) (cartoes.Invoice, error) {
	r.seen = ctx
	return cartoes.Invoice{}, errors.New("detalhe interno ficticio")
}

func TestPaymentSafeErrorAndContext(t *testing.T) {
	repo := &failingPaymentRepository{}
	h := NewHandlerWithRepositories(contas.NewMemoryRepository(), cartoes.NewMemoryRepository(), repo, "teste")
	r := httptest.NewRequest("POST", "/api/v1/cards/1/invoices/1/payment", strings.NewReader(`{"account_id":"1","date":"2026-02-10"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if repo.seen != r.Context() || w.Code != 500 || strings.Contains(w.Body.String(), "detalhe") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestPaymentEndpoints(t *testing.T) {
	h := NewHandler()
	request(t, h, "POST", "/api/v1/accounts", `{"name":"Ficticia","initial_balance_cents":1000,"initial_balance_date":"2026-01-01"}`, 201)
	request(t, h, "POST", "/api/v1/cards", `{"name":"Ficticio"}`, 201)
	request(t, h, "POST", "/api/v1/cards/1/invoices", `{"start_date":"2026-01-01","closing_date":"2026-01-31","due_date":"2026-02-10"}`, 201)
	base := "/api/v1/cards/1/invoices/1"
	request(t, h, "POST", base+"/purchases", `{"description":"Ficticia","amount_cents":1234,"date":"2026-01-15"}`, 201)
	body := `{"account_id":"1","date":"2026-02-10"}`
	request(t, h, "POST", base+"/payment", body, 409)
	request(t, h, "POST", base+"/close", `{}`, 200)
	for _, invalid := range []string{`null`, `{}`, `{"account_id":1,"date":"2026-02-10"}`, `{"account_id":"1","date":"2026-02-30"}`, `{"account_id":"1","date":"2026-02-10","amount_cents":1}`, `{} {}`} {
		request(t, h, "POST", base+"/payment", invalid, 400)
	}
	request(t, h, "POST", base+"/payment", `{"account_id":"999","date":"2026-02-10"}`, 404)
	paid := request(t, h, "POST", base+"/payment", body, 200)
	if string(request(t, h, "POST", base+"/payment", body, 200)) != string(paid) {
		t.Fatal("resposta repetida mudou")
	}
	request(t, h, "POST", base+"/payment", `{"account_id":"1","date":"2026-02-11"}`, 409)
	var invoice cartoes.Invoice
	if err := json.Unmarshal(request(t, h, "GET", base, "", 200), &invoice); err != nil || invoice.Status != "paid" || invoice.Payment == nil || invoice.Payment.AmountCents != 1234 {
		t.Fatal(invoice, err)
	}
	var account contas.Account
	if err := json.Unmarshal(request(t, h, "GET", "/api/v1/accounts/1/balance", "", 200), &account); err != nil || account.BalanceCents != -234 {
		t.Fatal(account, err)
	}
	var entries []contas.Entry
	if err := json.Unmarshal(request(t, h, "GET", "/api/v1/accounts/1/entries", "", 200), &entries); err != nil || len(entries) != 1 || entries[0].Kind != "invoice_payment" || entries[0].InvoiceID != "1" {
		t.Fatal(entries, err)
	}
	request(t, h, "POST", "/api/v1/accounts/1/entries", `{"kind":"invoice_payment","description":"Ficticia","amount_cents":1,"date":"2026-02-10"}`, 400)
}
