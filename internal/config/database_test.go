package config

import (
	"crypto/rand"
	"net/url"
	"strings"
	"testing"
)

func TestDatabaseVariables(t *testing.T) {
	// Valor descartável gerado em execução; não usa credenciais reais.
	password := rand.Text() + "@:/?# %"
	env := map[string]string{"DB_HOST": "127.0.0.1", "DB_USER": "usuario_ficticio", "DB_PASSWORD": password}
	get := func(k string) string { return env[k] }
	raw, err := databaseURL(get)
	if err != nil {
		t.Fatal("configuração válida rejeitada")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal("URL gerada inválida")
	}
	got, _ := u.User.Password()
	if got != password || u.User.Username() != env["DB_USER"] || u.Host != "127.0.0.1:5432" || u.Path != "/financeiro_go" || u.Query().Get("sslmode") != "require" {
		t.Fatal("configuração não preservada")
	}
	env["DB_NAME"] = "financeiro_go_test"
	env["DB_SSLMODE"] = "disable"
	env["DB_PORT"] = "5433"
	env["DB_HOST"] = "::1"
	raw, err = databaseURL(get)
	if err != nil {
		t.Fatal("configuração alternativa rejeitada")
	}
	u, err = url.Parse(raw)
	if err != nil || u.Host != "[::1]:5433" || u.Path != "/financeiro_go_test" || u.Query().Get("sslmode") != "disable" {
		t.Fatal("configuração alternativa incorreta")
	}
	env["PERSISTENCE"] = "postgres"
	cfg, err := load(get)
	if err != nil || cfg.DatabaseURL != raw {
		t.Fatal("API não usa DB_*")
	}
}

func TestDatabaseVariablesValidation(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"DB_HOST", ""}, {"DB_USER", " "}, {"DB_PASSWORD", ""},
		{"DB_PORT", "0"}, {"DB_PORT", "65536"}, {"DB_PORT", "invalid"},
		{"DB_NAME", "outro_banco"}, {"DB_SSLMODE", "invalid"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			password := rand.Text()
			env := map[string]string{"DB_HOST": "127.0.0.1", "DB_USER": "usuario_ficticio", "DB_PASSWORD": password}
			env[tc.key] = tc.value
			_, err := databaseURL(func(k string) string { return env[k] })
			if err == nil || strings.Contains(err.Error(), password) {
				t.Fatal("validação ausente ou exposição de segredo")
			}
		})
	}
}

func TestExplicitURLPrecedence(t *testing.T) {
	raw := "postgresql://localhost/financeiro_go"
	got, err := databaseURL(func(k string) string {
		if k == "FINANCEIRO_DATABASE_URL" {
			return raw
		}
		return "invalid"
	})
	if err != nil || got != raw {
		t.Fatal("URL explícita deve ter precedência")
	}
}
