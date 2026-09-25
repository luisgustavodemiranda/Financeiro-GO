package server

import (
	"financeirogo/internal/cartoes"
	"net/http"
)

func registerCardRoutes(mux *http.ServeMux, service *cartoes.Service) {
	mux.HandleFunc("POST /api/v1/cards", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name string `json:"name"`
		}
		if !decode(w, r, &input) {
			return
		}
		card, err := service.Create(r.Context(), input.Name)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, card)
	})
	mux.HandleFunc("GET /api/v1/cards", func(w http.ResponseWriter, r *http.Request) {
		cards, err := service.List(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, cards)
	})
}
