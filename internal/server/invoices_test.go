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

func TestInvoiceEndpoints(t *testing.T) {
	h := NewHandler()
	request(t, h, "POST", "/api/v1/accounts", `{"name":"Ficticia","initial_balance_cents":1000,"initial_balance_date":"2026-01-01"}`, 201)
	before := string(request(t, h, "GET", "/api/v1/accounts/1/balance", "", 200))
	request(t, h, "POST", "/api/v1/cards", `{"name":"Ficticio"}`, 201)
	base := "/api/v1/cards/1/invoices"
	if string(request(t, h, "GET", base, "", 200)) != "[]\n" {
		t.Fatal("lista vazia")
	}
	body := `{"start_date":"2026-01-01","closing_date":"2026-01-31","due_date":"2026-02-10"}`
	request(t, h, "POST", base, body, 201)
	request(t, h, "POST", base, body, 409)
	request(t, h, "POST", "/api/v1/cards/999/invoices", body, 404)
	for _, invalid := range []string{`null`, `{}`, `{"start_date":"2026-02-01","closing_date":"2026-02-30","due_date":"2026-03-10"}`, `{"start_date":123}`, `{"unknown":1}`} {
		request(t, h, "POST", base, invalid, 400)
	}
	path := base + "/1"
	for _, invalid := range []string{`null`, `{}`, `{"description":"Ficticia","amount_cents":1.5,"date":"2026-01-10"}`, `{"description":"Ficticia","amount_cents":9223372036854775808,"date":"2026-01-10"}`, `{"description":"Ficticia","amount_cents":"1","date":"2026-01-10"}`, `{"description":"Ficticia","amount_cents":0,"date":"2026-01-10"}`, `{"description":"Ficticia","amount_cents":1,"date":"2026-02-01"}`, `{"extra":1}`} {
		request(t, h, "POST", path+"/purchases", invalid, 400)
	}
	purchase := `{"description":"Compra ficticia","amount_cents":9007199254740993,"date":"2026-01-31"}`
	response := request(t, h, "POST", path+"/purchases", purchase, 201)
	if !strings.Contains(string(response), `"total_cents":9007199254740993`) {
		t.Fatal("centavos perderam precisão")
	}
	var invoice cartoes.Invoice
	if err := json.Unmarshal(request(t, h, "GET", path, "", 200), &invoice); err != nil || len(invoice.Purchases) != 1 || invoice.TotalCents != 9007199254740993 {
		t.Fatal(invoice, err)
	}
	request(t, h, "GET", base, "", 200)
	request(t, h, "GET", "/api/v1/cards/2/invoices/1", "", 404)
	for _, invalid := range []string{`null`, `{"status":"open"}`, `{} {}`} {
		request(t, h, "POST", path+"/close", invalid, 400)
	}
	closed := string(request(t, h, "POST", path+"/close", `{}`, 200))
	if again := string(request(t, h, "POST", path+"/close", `{}`, 200)); again != closed {
		t.Fatal("fechamento repetido alterou resultado")
	}
	request(t, h, "POST", path+"/purchases", purchase, 409)
	if string(request(t, h, "GET", "/api/v1/accounts/1/balance", "", 200)) != before {
		t.Fatal("cartão alterou saldo bancário")
	}
	if string(request(t, h, "GET", "/api/v1/accounts/1/entries", "", 200)) != "[]\n" {
		t.Fatal("compra criou lançamento bancário")
	}
	request(t, NewHandler(), "GET", path, "", 404)
}

type failingInvoiceRepository struct {
	cartoes.Repository
	seen context.Context
}

func (r *failingInvoiceRepository) GetInvoice(ctx context.Context, _, _ string) (cartoes.Invoice, error) {
	r.seen = ctx
	return cartoes.Invoice{}, errors.New("detalhe interno ficticio")
}

func TestInvoiceSafeErrorAndContext(t *testing.T) {
	repo := &failingInvoiceRepository{}
	h := NewHandlerWithRepositories(contas.NewMemoryRepository(), repo, "teste")
	r := httptest.NewRequest("GET", "/api/v1/cards/1/invoices/1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 500 || strings.Contains(w.Body.String(), "detalhe") || repo.seen != r.Context() {
		t.Fatal(w.Code, w.Body.String())
	}
}
