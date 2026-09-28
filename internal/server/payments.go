package server

import (
	"financeirogo/internal/cartoes"
	"net/http"
)

func registerPaymentRoutes(mux *http.ServeMux, service *cartoes.PaymentService) {
	mux.HandleFunc("POST /api/v1/cards/{card}/invoices/{invoice}/payment", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			AccountID string `json:"account_id"`
			Date      string `json:"date"`
		}
		if !decode(w, r, &input) {
			return
		}
		invoice, err := service.Pay(r.Context(), r.PathValue("card"), r.PathValue("invoice"), input.AccountID, input.Date)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, invoice)
	})
}
