package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// LoadDatabaseURL compartilha a configuração entre a API e o comando de migration.
// O resultado contém segredo e nunca deve ser impresso ou registrado em log.
func LoadDatabaseURL() (string, error) { return databaseURL(os.Getenv) }

func databaseURL(get func(string) string) (string, error) {
	// A URL explícita tem precedência; não misturamos duas configurações.
	if raw := get("FINANCEIRO_DATABASE_URL"); raw != "" {
		return raw, nil
	}
	value := func(key, fallback string) string {
		if v := get(key); v != "" {
			return v
		}
		return fallback
	}
	host := strings.TrimSpace(get("DB_HOST"))
	user := get("DB_USER")
	password := get("DB_PASSWORD")
	if host == "" || strings.TrimSpace(user) == "" || password == "" {
		return "", fmt.Errorf("configure FINANCEIRO_DATABASE_URL ou DB_HOST, DB_USER e DB_PASSWORD")
	}
	port, err := strconv.Atoi(value("DB_PORT", "5432"))
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("DB_PORT deve estar entre 1 e 65535")
	}
	name := value("DB_NAME", "financeiro_go")
	if name != "financeiro_go" && name != "financeiro_go_test" {
		return "", fmt.Errorf("DB_NAME deve ser financeiro_go ou financeiro_go_test")
	}
	sslmode := value("DB_SSLMODE", "require")
	switch sslmode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return "", fmt.Errorf("DB_SSLMODE inválido")
	}
	// url.UserPassword escapa caracteres especiais sem concatenar credenciais.
	u := url.URL{Scheme: "postgresql", Host: net.JoinHostPort(host, strconv.Itoa(port)), Path: "/" + name, User: url.UserPassword(user, password)}
	query := url.Values{"sslmode": []string{sslmode}}
	u.RawQuery = query.Encode()
	return u.String(), nil
}
