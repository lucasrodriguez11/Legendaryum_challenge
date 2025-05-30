package main

import (
	"legendaryum/internal/auth"
	"legendaryum/internal/config"
	"legendaryum/internal/middleware"
	"legendaryum/internal/tasks"
	"legendaryum/pkg/database"
	"legendaryum/pkg/models"
	"log"

	"github.com/gofiber/fiber/v2"
	// "github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// @title           Legendaryum Task Management API
// @version         1.0
// @description     API RESTful para gestión de tareas desarrollada en Go.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Cargar configuración
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error cargando configuración: %v", err)
	}

	// Conectar a la base de datos
	db := database.NewPostgres(cfg)

	// Ejecutar migraciones
	database.RunMigrations(cfg)

	// Migrar modelos
	if err := db.AutoMigrate(&models.User{}, &models.Task{}); err != nil {
		log.Fatalf("Error migrando modelos: %v", err)
	}

	// Crear aplicación Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		},
	})

	// Middleware globales
	app.Use(recover.New())
	app.Use(logger.New())

	app.Use(middleware.CORSMiddleware())
	app.Use(middleware.SwaggerUI())

	// Handlers
	authHandler := auth.NewHandler(db, cfg)
	taskHandler := tasks.NewHandler(db, cfg)

	// Rutas públicas
	authGroup := app.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)

	// Rutas protegidas
	tasksGroup := app.Group("/tasks", middleware.AuthMiddleware(cfg))
	tasksGroup.Post("/", taskHandler.Create)
	tasksGroup.Get("/", taskHandler.List)
	tasksGroup.Get("/:id", taskHandler.Get)
	tasksGroup.Put("/:id", taskHandler.Update)
	tasksGroup.Delete("/:id", taskHandler.Delete)

	// Ruta de salud
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Legendaryum API is running",
		})
	})

	// Iniciar servidor
	log.Printf("Servidor iniciado en el puerto %s", cfg.Port)
	log.Printf("Documentación Swagger disponible en: http://localhost:%s/docs", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
