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

	"legendaryum/internal/auth"
	"legendaryum/internal/config"
	"legendaryum/pkg/models"
	"legendaryum/pkg/utils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAuthLogin(t *testing.T) {
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

	// Generar datos únicos para evitar conflictos
	timestamp := time.Now().UnixNano()
	testEmail := fmt.Sprintf("login_%d@example.com", timestamp)
	testPassword := "testlogin123"
	t.Logf("📧 Email de prueba: %s", testEmail)

	//  INICIAR TRANSACCIÓN
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
	h := auth.NewHandler(tx, cfg)
	app.Post("/auth/login", h.Login)
	t.Log("🎯 Handler de login configurado con transacción")

	//  CREAR USUARIO DE PRUEBA DENTRO DE LA TRANSACCIÓN
	hash, err := utils.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("❌ No se pudo hashear la contraseña: %v", err)
	}
	t.Log("🔐 Contraseña hasheada correctamente")

	testUser := &models.User{
		FirstName:    "Test",
		LastName:     "Login",
		Email:        testEmail,
		PasswordHash: hash,
	}

	if err := tx.Create(testUser).Error; err != nil {
		t.Fatalf("❌ No se pudo crear el usuario de prueba: %v", err)
	}
	t.Logf("👤 Usuario de prueba creado - ID: %s", testUser.ID)

	// Verificar que el usuario existe en la transacción
	var createdUser models.User
	result := tx.Where("email = ?", testEmail).First(&createdUser)
	assert.NoError(t, result.Error, "El usuario debería existir en la transacción")
	t.Log("✅ Usuario verificado en la transacción")

	//  CASO 1: LOGIN EXITOSO
	t.Log("🧪 Probando caso 1: Login exitoso")
	payload := map[string]interface{}{
		"email":    testEmail,
		"password": testPassword,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("❌ No se pudo serializar el payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err, "No debería haber error en la request")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Login exitoso debería retornar 200")

	if resp.StatusCode == http.StatusOK {
		t.Log("✅ Caso 1 exitoso: Login con credenciales correctas")
	} else {
		t.Logf("❌ Caso 1 falló: Status code %d", resp.StatusCode)
	}

	//  CASO 2: CONTRASEÑA INCORRECTA
	t.Log("🧪 Probando caso 2: Contraseña incorrecta")
	payload["password"] = "wrongpassword123"
	body, _ = json.Marshal(payload)

	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req, -1)
	assert.NoError(t, err, "No debería haber error en la request")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Contraseña incorrecta debería retornar 401")

	if resp.StatusCode == http.StatusUnauthorized {
		t.Log("✅ Caso 2 exitoso: Contraseña incorrecta rechazada")
	} else {
		t.Logf("❌ Caso 2 falló: Status code %d", resp.StatusCode)
	}

	//  CASO 3: EMAIL NO REGISTRADO
	t.Log("🧪 Probando caso 3: Email no registrado")
	payload["email"] = fmt.Sprintf("noexiste_%d@example.com", timestamp)
	payload["password"] = testPassword
	body, _ = json.Marshal(payload)

	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req, -1)
	assert.NoError(t, err, "No debería haber error en la request")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Email no registrado debería retornar 401")

	if resp.StatusCode == http.StatusUnauthorized {
		t.Log("✅ Caso 3 exitoso: Email no registrado rechazado")
	} else {
		t.Logf("❌ Caso 3 falló: Status code %d", resp.StatusCode)
	}

	//  CASO 4: DATOS VACÍOS
	t.Log("🧪 Probando caso 4: Datos vacíos")
	payload = map[string]interface{}{
		"email":    "",
		"password": "",
	}
	body, _ = json.Marshal(payload)

	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req, -1)
	assert.NoError(t, err, "No debería haber error en la request")
	assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized,
		"Datos vacíos deberían retornar 400 o 401")

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		t.Log("✅ Caso 4 exitoso: Datos vacíos rechazados")
	} else {
		t.Logf("❌ Caso 4 falló: Status code %d", resp.StatusCode)
	}

	//  CASO 5: JSON MALFORMADO
	t.Log("🧪 Probando caso 5: JSON malformado")
	malformedJSON := `{"email": "test@test.com", "password": }`

	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(malformedJSON)))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req, -1)
	assert.NoError(t, err, "No debería haber error en la request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "JSON malformado debería retornar 400")

	if resp.StatusCode == http.StatusBadRequest {
		t.Log("✅ Caso 5 exitoso: JSON malformado rechazado")
	} else {
		t.Logf("❌ Caso 5 falló: Status code %d", resp.StatusCode)
	}

	t.Log("🎉 Todos los casos de test completados")

	// Verificar que el usuario aún existe en la transacción antes del rollback
	var finalUser models.User
	result = tx.Where("email = ?", testEmail).First(&finalUser)
	assert.NoError(t, result.Error, "El usuario debería seguir existiendo en la transacción")
	t.Log("✅ Usuario aún existe en la transacción antes del rollback")
}

// Test adicional para verificar limpieza específica del login
func TestLoginDatabaseCleanup(t *testing.T) {
	//  CONECTAR A LA BASE DE DATOS REAL
	dsn := "host=localhost user=postgres password=postgres dbname=legendaryum_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("❌ No se pudo conectar a la base de datos: %v", err)
	}

	//  USAR TRANSACCIÓN TAMBIÉN AQUÍ
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("❌ No se pudo iniciar la transacción: %v", tx.Error)
	}
	defer tx.Rollback()

	// Verificar que no hay usuarios de login en la base de datos real
	var count int64
	tx.Model(&models.User{}).Where("email LIKE ?", "login_%@example.com").Count(&count)

	assert.Equal(t, int64(0), count, "No debería haber usuarios de login en la base de datos real")
	if count == 0 {
		t.Log("✅ Base de datos limpia - no hay usuarios de login residuales")
	} else {
		t.Logf("⚠️  Encontrados %d usuarios de login en la base de datos real", count)

		// Si hay usuarios residuales, mostrarlos
		var users []models.User
		tx.Where("email LIKE ?", "login_%@example.com").Find(&users)
		for _, u := range users {
			t.Logf("   - Usuario residual: ID=%s, Email=%s, Nombre=%s %s",
				u.ID, u.Email, u.FirstName, u.LastName)
		}
	}

	t.Log("🧹 Verificación de limpieza completada")
}
