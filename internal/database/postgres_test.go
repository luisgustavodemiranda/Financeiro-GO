package database

import (
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		valid     bool
	}{
		{"empty", "", false},
		{"invalid", "://exemplo", false},
		{"other database", "postgresql://localhost/outro_banco", false},
		{"no database", "postgresql://localhost/", false},
		{"application", "postgresql://localhost/financeiro_go", true},
		{"test", "host=localhost dbname=financeiro_go_test", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseConfig(tc.raw)
			if (err == nil) != tc.valid {
				t.Fatalf("resultado inesperado: %v", err)
			}
			if tc.valid && (cfg.MaxConns != 5 || cfg.ConnConfig.RuntimeParams["application_name"] != "financeiro-go") {
				t.Fatal("configuração incorreta")
			}
		})
	}
}

func TestParseErrorDoesNotExposeInput(t *testing.T) {
	input := "entrada-ficticia-invalida"
	_, err := ParseConfig(input)
	if err == nil || strings.Contains(err.Error(), input) {
		t.Fatal("erro deve omitir o conteúdo da configuração")
	}
}
