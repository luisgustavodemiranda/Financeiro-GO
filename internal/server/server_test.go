package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutes(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		code         int
	}{
		{"GET", "/health", 200}, {"GET", "/api/v1/status", 200}, {"POST", "/health", 405}, {"GET", "/unknown", 404},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			NewHandler().ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.code {
				t.Fatalf("status: %d", w.Code)
			}
			if w.Code == http.StatusOK {
				var got map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if tc.path == "/api/v1/status" && got["persistence"] != "memoria; dados perdidos ao reiniciar" {
					t.Fatal("estado incorreto")
				}
			}
		})
	}
}
