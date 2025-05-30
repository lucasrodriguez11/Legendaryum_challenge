package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	// Importa tus paquetes internos
	"legendaryum/internal/auth"
	"legendaryum/internal/config"
	"legendaryum/pkg/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAuthRegister(t *testing.T) {
	// Setup de la aplicación Fiber
	app := fiber.New()

	// Cargar configuración del .env
	cfg, err := config.Load()
	if nil != err {
		t.Fatalf("❌ No se pudo cargar la configuración: %v", err)
	}

	// Configuración para el test (JWT todavía hardcodeado, pero DB del .env)
	// cfg := &config.Config{
	// 	JWTSecret: "testsecretkey12345678901234567890",
	// 	JWTExpiry: "24h",
	// }

	// Conexión a la base de datos real usando configuración del .env
	// dsn := "host=localhost user=postgres password=postgres dbname=legendaryum_db port=5432 sslmode=disable"
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("❌ No se pudo conectar a la base de datos: %v", err)
	}
	t.Log("✅ Conexión a base de datos establecida")

	// Migración automática de tablas
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("❌ No se pudo migrar el modelo User: %v", err)
	}
	t.Log("✅ Migración de tablas completada")

	// Generar email único para evitar conflictos
	timestamp := time.Now().UnixNano()
	testEmail := fmt.Sprintf("test_%d@example.com", timestamp)
	t.Logf("📧 Email de prueba: %s", testEmail)

	//  INICIAR TRANSACCIÓN AQUÍ
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("❌ No se pudo iniciar la transacción: %v", tx.Error)
	}
	t.Log("🔄 Transacción iniciada")

	// Asegurar rollback en caso de panic
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			t.Logf("🚨 Panic detectado, rollback ejecutado: %v", r)
		}
	}()

	// Rollback automático al final del test
	defer func() {
		if err := tx.Rollback().Error; err != nil {
			t.Logf("⚠️ Error en rollback: %v", err)
		} else {
			t.Log("🔙 Rollback ejecutado - base de datos limpia")
		}
	}()

	//  CONFIGURAR HANDLER CON LA TRANSACCIÓN
	h := auth.NewHandler(tx, cfg) // Usar 'tx' en lugar de 'db'
	app.Post("/auth/register", h.Register)
	t.Log("🎯 Handler configurado con transacción")

	// Preparar payload del test
	payload := map[string]interface{}{
		"first_name": "Lucas Nahuel",
		"last_name":  "Rodriguez",
		"email":      testEmail,
		"password":   "test1230",
	}
	body, _ := json.Marshal(payload)
	t.Log("📦 Payload preparado")

	//  CREAR REQUEST HTTP CON HTTPTEST
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	t.Log("📨 Request HTTP creado con httptest")

	//  EJECUTAR EL TEST CON HTTPTEST
	t.Log("🚀 Ejecutando request...")
	resp, err := app.Test(req, -1) // app.Test usa httptest internamente
	if err != nil {
		t.Fatalf("❌ Error ejecutando request: %v", err)
	}

	// Verificar status code
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "El status code debería ser 201 Created")
	if resp.StatusCode == http.StatusCreated {
		t.Log("✅ Status code correcto: 201 Created")
	} else {
		t.Logf("❌ Status code incorrecto: %d", resp.StatusCode)
	}

	// Verificar que el usuario se creó en la base de datos (dentro de la transacción)
	var user models.User
	result := tx.Where("email = ?", testEmail).First(&user)

	// Assertions detalladas
	assert.NoError(t, result.Error, "El usuario debería existir en la base de datos")
	if result.Error == nil {
		t.Log("✅ Usuario encontrado en la base de datos (dentro de transacción)")

		assert.Equal(t, "Lucas Nahuel", user.FirstName, "El nombre debería coincidir")
		assert.Equal(t, "Rodriguez", user.LastName, "El apellido debería coincidir")
		assert.Equal(t, testEmail, user.Email, "El email debería coincidir")

		// Verificar que la contraseña fue hasheada
		assert.NotEqual(t, "test1230", user.PasswordHash, "La contraseña debería estar hasheada")
		assert.NotEmpty(t, user.PasswordHash, "La contraseña hasheada no debería estar vacía")

		t.Logf("✅ Datos del usuario verificados:")
		t.Logf("   - ID=%s", user.ID)
		t.Logf("   - Nombre: %s", user.FirstName)
		t.Logf("   - Apellido: %s", user.LastName)
		t.Logf("   - Email: %s", user.Email)
		t.Logf("   - Password hasheada: %s", user.PasswordHash[:20]+"...")
	} else {
		t.Logf("❌ Usuario no encontrado: %v", result.Error)
	}

	t.Log("🎉 Test completado exitosamente")
}

// Test adicional para verificar que la limpieza funciona
func TestDatabaseCleanup(t *testing.T) {
	//  CONECTAR A LA BASE DE DATOS REAL (SIN TRANSACCIÓN)
	dsn := "host=localhost user=postgres password=postgres dbname=legendaryum_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("❌ No se pudo conectar a la base de datos: %v", err)
	}

	//  INICIAR TRANSACCIÓN PARA ESTE TEST TAMBIÉN
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("❌ No se pudo iniciar la transacción: %v", tx.Error)
	}
	defer tx.Rollback()

	// Verificar que no hay usuarios de test en la base de datos real
	var count int64
	tx.Model(&models.User{}).Where("email LIKE ?", "test_%@example.com").Count(&count)

	assert.Equal(t, int64(0), count, "No debería haber usuarios de test en la base de datos real")
	if count == 0 {
		t.Log("✅ Base de datos limpia - no hay usuarios de test residuales")
	} else {
		t.Logf("⚠️  Encontrados %d usuarios de test en la base de datos real", count)

		// Si hay usuarios residuales, mostrarlos
		var users []models.User
		tx.Where("email LIKE ?", "test_%@example.com").Find(&users)
		for _, u := range users {
			t.Logf("   - Usuario residual: ID=%s, Email=%s", u.ID, u.Email)
		}
	}
}
