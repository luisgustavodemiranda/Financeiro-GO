package server

import (
	"financeirogo/internal/cartoes"
	"net/http"
)

func registerInvoiceRoutes(mux *http.ServeMux, service *cartoes.Service) {
	const base = "/api/v1/cards/{card}/invoices"
	mux.HandleFunc("POST "+base, func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			StartDate   string `json:"start_date"`
			ClosingDate string `json:"closing_date"`
			DueDate     string `json:"due_date"`
		}
		if !decode(w, r, &input) {
			return
		}
		invoice, err := service.CreateInvoice(r.Context(), r.PathValue("card"), input.StartDate, input.ClosingDate, input.DueDate)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, invoice)
	})
	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		invoices, err := service.ListInvoices(r.Context(), r.PathValue("card"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, invoices)
	})
	mux.HandleFunc("GET "+base+"/{invoice}", func(w http.ResponseWriter, r *http.Request) {
		invoice, err := service.GetInvoice(r.Context(), r.PathValue("card"), r.PathValue("invoice"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, invoice)
	})
	mux.HandleFunc("POST "+base+"/{invoice}/purchases", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Description string `json:"description"`
			AmountCents int64  `json:"amount_cents"`
			Date        string `json:"date"`
		}
		if !decode(w, r, &input) {
			return
		}
		invoice, err := service.RegisterPurchase(r.Context(), r.PathValue("card"), r.PathValue("invoice"), input.Description, input.AmountCents, input.Date)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, invoice)
	})
	mux.HandleFunc("POST "+base+"/{invoice}/close", func(w http.ResponseWriter, r *http.Request) {
		var input *struct{}
		if !decode(w, r, &input) {
			return
		}
		if input == nil {
			writeError(w, cartoes.ErrInvoiceInvalid)
			return
		}
		invoice, err := service.CloseInvoice(r.Context(), r.PathValue("card"), r.PathValue("invoice"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, invoice)
	})
}
