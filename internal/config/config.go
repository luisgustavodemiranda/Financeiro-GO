package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Persistence     string
	DatabaseURL     string
	DatabaseTimeout time.Duration
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	return load(os.Getenv)
}

func load(get func(string) string) (Config, error) {
	value := func(key, fallback string) string {
		if v := get(key); v != "" {
			return v
		}
		return fallback
	}
	port, err := strconv.Atoi(value("HTTP_PORT", "8081"))
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("HTTP_PORT deve estar entre 1 e 65535")
	}
	cfg := Config{Address: net.JoinHostPort(value("HTTP_HOST", "127.0.0.1"), strconv.Itoa(port))}
	cfg.Persistence = value("PERSISTENCE", "memory")
	if cfg.Persistence != "memory" && cfg.Persistence != "postgres" {
		return Config{}, fmt.Errorf("PERSISTENCE deve ser memory ou postgres")
	}
	if cfg.Persistence == "postgres" {
		var err error
		cfg.DatabaseURL, err = databaseURL(get)
		if err != nil {
			return Config{}, err
		}
	}
	for _, item := range []struct {
		key  string
		dest *time.Duration
	}{
		{"HTTP_READ_TIMEOUT", &cfg.ReadTimeout}, {"HTTP_WRITE_TIMEOUT", &cfg.WriteTimeout}, {"HTTP_SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout},
		{"DATABASE_TIMEOUT", &cfg.DatabaseTimeout},
	} {
		d, err := time.ParseDuration(value(item.key, "10s"))
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("%s deve ser uma duração positiva", item.key)
		}
		*item.dest = d
	}
	return cfg, nil
}
