package config

import (
	"os"
)

// Config representa la configuración de la aplicación
type Config struct {
	Port      string
	JWTSecret string
	JWTExpiry string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
}

// Load carga la configuración desde variables de entorno
func Load() (*Config, error) {
	cfg := &Config{
		Port:      getEnv("PORT", "8080"),
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpiry: getEnv("JWT_EXPIRY", "24h"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "postgres"),
		DBPass:    getEnv("DB_PASS", "postgres"),
		DBName:    getEnv("DB_NAME", "legendaryum_db"),
	}

	return cfg, nil
}

// getEnv obtiene una variable de entorno o un valor por defecto
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
