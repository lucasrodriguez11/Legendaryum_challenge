package middleware

import (
	"legendaryum/internal/config"
	"legendaryum/pkg/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware valida el token JWT y extrae el ID del usuario
func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Obtener el token del header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token no proporcionado",
			})
		}

		// Verificar que el header tenga el formato correcto
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Formato de token inválido",
			})
		}

		// Validar el token
		claims, err := utils.ValidateJWT(parts[1], cfg.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token inválido o expirado",
			})
		}

		// Guardar el ID del usuario en el contexto
		c.Locals("user_id", claims.UserID)

		return c.Next()
	}
}

// Middleware de autenticación
