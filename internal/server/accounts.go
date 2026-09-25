package server

import (
	"encoding/json"
	"errors"
	"financeirogo/internal/contas"
	"io"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "erro interno"
	switch {
	case errors.Is(err, contas.ErrNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, contas.ErrInvalid):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, contas.ErrOverflow):
		status = http.StatusConflict
		message = err.Error()
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err := d.Decode(value)
	if err == nil {
		err = d.Decode(new(any))
		if err == io.EOF {
			return true
		}
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "envie um único objeto JSON válido, com campos conhecidos e centavos inteiros int64"})
	return false
}

func registerAccountRoutes(mux *http.ServeMux, service *contas.Service) {
	mux.HandleFunc("POST /api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name                string `json:"name"`
			InitialBalanceCents *int64 `json:"initial_balance_cents"`
			InitialBalanceDate  string `json:"initial_balance_date"`
		}
		if !decode(w, r, &input) {
			return
		}
		if input.InitialBalanceCents == nil {
			writeError(w, contas.ErrInvalid)
			return
		}
		account, err := service.Create(r.Context(), input.Name, *input.InitialBalanceCents, input.InitialBalanceDate)
		if err != nil {
			writeError(w, err)
			return
		}
		w.Header().Set("Location", "/api/v1/accounts/"+account.ID+"/balance")
		writeJSON(w, http.StatusCreated, account)
	})
	mux.HandleFunc("GET /api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		accounts, err := service.List(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, accounts)
	})
	mux.HandleFunc("POST /api/v1/accounts/{id}/entries", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Kind        string `json:"kind"`
			Description string `json:"description"`
			AmountCents int64  `json:"amount_cents"`
			Date        string `json:"date"`
		}
		if !decode(w, r, &input) {
			return
		}
		entry, err := service.Register(r.Context(), r.PathValue("id"), input.Kind, input.Description, input.AmountCents, input.Date)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, entry)
	})
	mux.HandleFunc("GET /api/v1/accounts/{id}/entries", func(w http.ResponseWriter, r *http.Request) {
		state, err := service.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, state.Entries)
	})
	mux.HandleFunc("GET /api/v1/accounts/{id}/balance", func(w http.ResponseWriter, r *http.Request) {
		state, err := service.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, state.Account)
	})
}
