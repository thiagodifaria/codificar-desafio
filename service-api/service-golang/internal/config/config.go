package config

import (
	"fmt"
	"os"
)

// Config reúne as configurações necessárias para executar a API.
type Config struct {
	Address     string
	DatabaseURL string
}

// Load carrega as configurações do ambiente e aplica valores adequados ao desenvolvimento local.
func Load() Config {
	return Config{
		Address:     ":" + envOrDefault("APP_PORT", "8080"),
		DatabaseURL: envOrDefault("DATABASE_URL", "postgres://chamados:chamados@localhost:5432/chamados?sslmode=disable"),
	}
}

// Validate verifica se as configurações obrigatórias foram informadas.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	return nil
}

// envOrDefault retorna o valor da variável de ambiente ou utiliza o fallback informado.
func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
