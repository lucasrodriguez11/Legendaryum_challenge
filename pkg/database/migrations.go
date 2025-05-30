package database

import (
	"fmt"
	"log"

	"legendaryum/internal/config"

	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Gestión de migracionesok, continuemos con lo recomendado

// RunMigrations ejecuta las migraciones de la carpeta ./migrations
func RunMigrations(cfg *config.Config) {
	url := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPass,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	m, err := migrate.New(
		"file://./migrations",
		url,
	)
	if err != nil {
		log.Fatalf("No se pudo inicializar migrate: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Error al aplicar migraciones: %v", err)
	}
	log.Println("Migraciones aplicadas correctamente")
}
