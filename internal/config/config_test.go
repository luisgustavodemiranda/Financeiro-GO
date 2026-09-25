package config

import "testing"

func TestLoad(t *testing.T) {
	for _, tc := range []struct {
		name    string
		env     map[string]string
		invalid bool
	}{
		{"defaults", nil, false}, {"custom", map[string]string{"HTTP_PORT": "9090", "HTTP_READ_TIMEOUT": "2s"}, false},
		{"invalid port", map[string]string{"HTTP_PORT": "abc"}, true}, {"out of range", map[string]string{"HTTP_PORT": "65536"}, true},
		{"zero timeout", map[string]string{"HTTP_READ_TIMEOUT": "0s"}, true}, {"invalid timeout", map[string]string{"HTTP_WRITE_TIMEOUT": "abc"}, true},
		{"invalid persistence", map[string]string{"PERSISTENCE": "invalid"}, true},
		{"missing database", map[string]string{"PERSISTENCE": "postgres"}, true},
		{"postgres", map[string]string{"PERSISTENCE": "postgres", "FINANCEIRO_DATABASE_URL": "postgresql://localhost/financeiro_go"}, false},
		{"database timeout", map[string]string{"DATABASE_TIMEOUT": "0s"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := load(func(k string) string { return tc.env[k] })
			if (err != nil) != tc.invalid {
				t.Fatalf("erro inesperado: %v", err)
			}
			if tc.name == "defaults" && cfg.Address != "127.0.0.1:8081" {
				t.Fatalf("endereço inesperado: %s", cfg.Address)
			}
			if tc.name == "defaults" && cfg.Persistence != "memory" {
				t.Fatal("modo padrão incorreto")
			}
			if tc.name == "postgres" && cfg.DatabaseURL != tc.env["FINANCEIRO_DATABASE_URL"] {
				t.Fatal("URL não repassada")
			}
		})
	}
}
