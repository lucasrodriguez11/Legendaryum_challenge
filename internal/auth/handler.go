package auth

import (
	"legendaryum/internal/config"
	"legendaryum/pkg/models"
	"legendaryum/pkg/utils"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Handler de autenticación

type Handler struct {
	DB     *gorm.DB
	Config *config.Config
}

func NewHandler(db *gorm.DB, cfg *config.Config) *Handler {
	return &Handler{DB: db, Config: cfg}
}

// Register godoc
// @Summary Registrar un nuevo usuario
// @Description Crea una nueva cuenta de usuario
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Datos de registro"
// @Success 201 {object} map[string]interface{} "Usuario creado exitosamente"
// @Failure 400 {object} map[string]interface{} "Error en los datos de entrada"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "JSON inválido"})
	}

	// Validaciones
	if len(strings.TrimSpace(req.FirstName)) < 2 || len(req.FirstName) > 50 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "El nombre debe tener entre 2 y 50 caracteres"})
	}
	if len(strings.TrimSpace(req.LastName)) < 2 || len(req.LastName) > 50 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "El apellido debe tener entre 2 y 50 caracteres"})
	}
	if !regexp.MustCompile(`^[a-zA-ZáéíóúÁÉÍÓÚüÜñÑ\s]+$`).MatchString(req.FirstName) || !regexp.MustCompile(`^[a-zA-ZáéíóúÁÉÍÓÚüÜñÑ\s]+$`).MatchString(req.LastName) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "Nombre y apellido solo pueden contener letras y espacios"})
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "Email inválido"})
	}
	if len(req.Password) < 6 || len(req.Password) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "La contraseña debe tener entre 6 y 100 caracteres"})
	}

	// Email único
	var count int64
	h.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "El email ya está registrado"})
	}

	// Hash de password
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Error al hashear la contraseña"})
	}

	now := time.Now().UTC()
	user := models.User{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Error al crear usuario"})
	}

	token, err := utils.GenerateJWT(user.ID, h.Config.JWTSecret, h.Config.JWTExpiry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Error al generar token"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user": fiber.Map{
				"id":         user.ID,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"email":      user.Email,
				"created_at": user.CreatedAt,
			},
			"token": token,
		},
	})
}

// Login godoc
// @Summary Iniciar sesión
// @Description Autentica a un usuario y devuelve un token JWT
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Credenciales de inicio de sesión"
// @Success 200 {object} map[string]interface{} "Login exitoso"
// @Failure 400 {object} map[string]interface{} "Error en los datos de entrada"
// @Failure 401 {object} map[string]interface{} "Credenciales inválidas"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "JSON inválido"})
	}
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "Email y contraseña son requeridos"})
	}
	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": "Credenciales inválidas"})
	}
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": "Credenciales inválidas"})
	}
	token, err := utils.GenerateJWT(user.ID, h.Config.JWTSecret, h.Config.JWTExpiry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Error al generar token"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user": fiber.Map{
				"id":         user.ID,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"email":      user.Email,
			},
			"token": token,
		},
	})
}
