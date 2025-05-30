package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"legendaryum/internal/auth"
	"legendaryum/internal/config"
	"legendaryum/internal/tasks"
	"legendaryum/pkg/models"
	"legendaryum/pkg/utils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestTasks(t *testing.T) {
	// Setup de la aplicación Fiber
	app := fiber.New()

	// Cargar configuración del .env
	cfg, err := config.Load()
	if nil != err {
		t.Fatalf("❌ No se pudo cargar la configuración: %v", err)
	}

	// Configuración para el test
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
	if err := db.AutoMigrate(&models.User{}, &models.Task{}); err != nil {
		t.Fatalf("❌ No se pudo migrar los modelos: %v", err)
	}
	t.Log("✅ Migración de tablas completada")

	// Generar datos únicos para evitar conflictos
	timestamp := time.Now().UnixNano()
	creatorEmail := fmt.Sprintf("creator_%d@example.com", timestamp)
	assigneeEmail := fmt.Sprintf("assignee_%d@example.com", timestamp)
	testPassword := "test123"
	t.Logf("📧 Emails de prueba: %s, %s", creatorEmail, assigneeEmail)

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

	//  CONFIGURAR HANDLERS CON LA TRANSACCIÓN
	authHandler := auth.NewHandler(tx, cfg)
	tasksHandler := tasks.NewHandler(tx, cfg)

	// Configurar rutas
	app.Post("/auth/login", authHandler.Login)

	//  CREAR USUARIOS DE PRUEBA DENTRO DE LA TRANSACCIÓN
	hash, err := utils.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("❌ No se pudo hashear la contraseña: %v", err)
	}
	t.Log("🔐 Contraseña hasheada correctamente")

	// Crear usuario creador
	creator := &models.User{
		FirstName:    "Test",
		LastName:     "Creator",
		Email:        creatorEmail,
		PasswordHash: hash,
	}
	if err := tx.Create(creator).Error; err != nil {
		t.Fatalf("❌ No se pudo crear el usuario creador: %v", err)
	}
	t.Logf("👤 Usuario creador creado - ID: %v", creator.ID)

	// Configurar middleware de autenticación
	app.Use(func(c *fiber.Ctx) error {
		// En los tests, simulamos que el usuario está autenticado
		c.Locals("user_id", creator.ID)
		return c.Next()
	})

	// Rutas de tareas
	app.Post("/tasks", tasksHandler.Create)
	app.Get("/tasks", tasksHandler.List)
	app.Get("/tasks/:id", tasksHandler.Get)
	app.Put("/tasks/:id", tasksHandler.Update)
	app.Delete("/tasks/:id", tasksHandler.Delete)

	t.Log("🎯 Handlers configurados con transacción")

	// Crear usuario asignado
	assignee := &models.User{
		FirstName:    "Test",
		LastName:     "Assignee",
		Email:        assigneeEmail,
		PasswordHash: hash,
	}
	if err := tx.Create(assignee).Error; err != nil {
		t.Fatalf("❌ No se pudo crear el usuario asignado: %v", err)
	}
	t.Logf("👤 Usuario asignado creado - ID: %v", assignee.ID)

	//  OBTENER TOKEN JWT PARA EL CREADOR CON MEJOR MANEJO DE ERRORES
	loginPayload := map[string]interface{}{
		"email":    creatorEmail,
		"password": testPassword,
	}
	loginBody, _ := json.Marshal(loginPayload)
	t.Logf("🔐 Intentando login con payload: %s", string(loginBody))

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := app.Test(loginReq, -1)
	assert.NoError(t, err, "No debería haber error en el login")

	// Leer el cuerpo completo de la respuesta para debugging
	body, err := io.ReadAll(loginResp.Body)
	assert.NoError(t, err, "Debería poder leer el cuerpo de la respuesta")
	t.Logf("📋 Respuesta completa del login (Status: %d): %s", loginResp.StatusCode, string(body))

	// Verificar el status code
	if loginResp.StatusCode != http.StatusOK {
		t.Logf("❌ Login falló con status %d", loginResp.StatusCode)

		// Intentar decodificar el error
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(body, &errorResponse); err == nil {
			t.Logf("❌ Error del servidor: %+v", errorResponse)
		}

		t.Fatalf("❌ Login debería ser exitoso pero recibió status %d", loginResp.StatusCode)
	}

	// Decodificar la respuesta
	var loginResponse map[string]interface{}
	err = json.Unmarshal(body, &loginResponse)
	assert.NoError(t, err, "No debería haber error al decodificar la respuesta")

	t.Logf("🔍 Estructura de loginResponse: %+v", loginResponse)

	// Verificar que existe el campo data
	dataValue, exists := loginResponse["data"]
	if !exists {
		t.Logf("❌ No se encontró campo 'data' en la respuesta")
		t.Logf("🔍 Campos disponibles: %+v", loginResponse)
		t.Fatal("❌ La respuesta debería contener un campo 'data'")
	}

	data, ok := dataValue.(map[string]interface{})
	if !ok {
		t.Logf("❌ El campo 'data' no es un objeto, es de tipo: %T", dataValue)
		t.Fatal("❌ El campo 'data' debería ser un objeto")
	}

	// Verificar que existe el token dentro de data
	tokenValue, exists := data["token"]
	if !exists {
		t.Logf("❌ No se encontró campo 'token' en data")
		t.Logf("🔍 Campos disponibles en data: %+v", data)
		t.Fatal("❌ El campo data debería contener un campo 'token'")
	}

	if tokenValue == nil {
		t.Logf("❌ El campo 'token' es nil")
		t.Fatal("❌ El token no debería ser nil")
	}

	token, ok := tokenValue.(string)
	if !ok {
		t.Logf("❌ El token no es una string, es de tipo: %T, valor: %+v", tokenValue, tokenValue)
		t.Fatal("❌ El token debería ser una string")
	}

	if token == "" {
		t.Fatal("❌ El token no debería estar vacío")
	}

	t.Logf("🔑 Token JWT obtenido: %s", token[:20]+"...")

	//  CASO 1: CREAR TAREA CON HTTPTEST
	t.Log("🧪 Probando caso 1: Crear tarea")
	taskPayload := map[string]interface{}{
		"title":       "Tarea de prueba automatizada",
		"description": "Descripción de la tarea de prueba creada con httptest",
		"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"priority":    "high",
		"status":      "pending",
		"assignee_id": assignee.ID,
	}
	taskBody, _ := json.Marshal(taskPayload)

	createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(taskBody))
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := app.Test(createReq, -1)
	assert.NoError(t, err, "No debería haber error en la creación")

	// Leer respuesta para debugging si hay error
	createBody, _ := io.ReadAll(createResp.Body)
	t.Logf("📋 Respuesta de creación (Status: %d): %s", createResp.StatusCode, string(createBody))

	assert.Equal(t, http.StatusCreated, createResp.StatusCode, "Creación debería ser exitosa")

	var createResponse map[string]interface{}
	err = json.Unmarshal(createBody, &createResponse)
	assert.NoError(t, err, "No debería haber error al decodificar la respuesta")
	assert.Equal(t, "success", createResponse["status"], "Debería recibir status success")

	taskData := createResponse["data"].(map[string]interface{})
	taskID := taskData["id"]
	t.Logf("✅ Tarea creada exitosamente - ID: %v", taskID)

	// Verificar que la tarea existe en la transacción
	var createdTask models.Task
	result := tx.Where("id = ?", taskID).First(&createdTask)
	assert.NoError(t, result.Error, "La tarea debería existir en la transacción")
	assert.Equal(t, "Tarea de prueba automatizada", createdTask.Title, "El título debería coincidir")
	t.Log("✅ Tarea verificada en la base de datos (transacción)")

	//  CASO 2: LISTAR TAREAS CON HTTPTEST
	t.Log("🧪 Probando caso 2: Listar tareas")
	listReq := httptest.NewRequest(http.MethodGet, "/tasks", nil)

	listResp, err := app.Test(listReq, -1)
	assert.NoError(t, err, "No debería haber error en el listado")
	assert.Equal(t, http.StatusOK, listResp.StatusCode, "Listado debería ser exitoso")

	var listResponse map[string]interface{}
	err = json.NewDecoder(listResp.Body).Decode(&listResponse)
	assert.NoError(t, err, "No debería haber error al decodificar la respuesta")
	assert.Equal(t, "success", listResponse["status"], "Debería recibir status success")

	tasks := listResponse["data"].([]interface{})
	assert.Greater(t, len(tasks), 0, "Debería haber al menos una tarea")
	t.Logf("✅ Listado de tareas exitoso - %d tareas encontradas", len(tasks))

	//  CASO 3: OBTENER TAREA ESPECÍFICA CON HTTPTEST
	t.Log("🧪 Probando caso 3: Obtener tarea específica")
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%v", taskID), nil)

	getResp, err := app.Test(getReq, -1)
	assert.NoError(t, err, "No debería haber error al obtener la tarea")
	assert.Equal(t, http.StatusOK, getResp.StatusCode, "Obtención debería ser exitosa")

	var getResponse map[string]interface{}
	err = json.NewDecoder(getResp.Body).Decode(&getResponse)
	assert.NoError(t, err, "No debería haber error al decodificar la respuesta")
	assert.Equal(t, "success", getResponse["status"], "Debería recibir status success")

	getTaskData := getResponse["data"].(map[string]interface{})
	assert.Equal(t, taskID, getTaskData["id"], "Debería obtener la tarea correcta")
	t.Log("✅ Obtención de tarea específica exitosa")

	//  CASO 4: ACTUALIZAR TAREA CON HTTPTEST
	t.Log("🧪 Probando caso 4: Actualizar tarea")
	updatePayload := map[string]interface{}{
		"title":       "Tarea actualizada con httptest",
		"description": "Descripción actualizada usando httptest",
		"status":      "in_progress",
		"priority":    "medium",
	}
	updateBody, _ := json.Marshal(updatePayload)

	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/tasks/%v", taskID), bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")

	updateResp, err := app.Test(updateReq, -1)
	assert.NoError(t, err, "No debería haber error en la actualización")
	assert.Equal(t, http.StatusOK, updateResp.StatusCode, "Actualización debería ser exitosa")

	var updateResponse map[string]interface{}
	err = json.NewDecoder(updateResp.Body).Decode(&updateResponse)
	assert.NoError(t, err, "No debería haber error al decodificar la respuesta")
	assert.Equal(t, "success", updateResponse["status"], "Debería recibir status success")

	updateTaskData := updateResponse["data"].(map[string]interface{})
	assert.Equal(t, "Tarea actualizada con httptest", updateTaskData["title"], "Debería actualizarse el título")
	assert.Equal(t, "in_progress", updateTaskData["status"], "Debería actualizarse el estado")
	t.Log("✅ Actualización de tarea exitosa")

	// Verificar actualización en la base de datos
	var updatedTask models.Task
	tx.Where("id = ?", taskID).First(&updatedTask)
	assert.Equal(t, "Tarea actualizada con httptest", updatedTask.Title, "El título debería estar actualizado en la DB")
	assert.Equal(t, "in_progress", updatedTask.Status, "El estado debería estar actualizado en la DB")
	t.Log("✅ Actualización verificada en la base de datos")

	//  CASO 5: ELIMINAR TAREA CON HTTPTEST
	t.Log("🧪 Probando caso 5: Eliminar tarea")
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/tasks/%v", taskID), nil)

	deleteResp, err := app.Test(deleteReq, -1)
	assert.NoError(t, err, "No debería haber error en la eliminación")
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode, "Eliminación debería ser exitosa")

	var deleteResponse map[string]interface{}
	err = json.NewDecoder(deleteResp.Body).Decode(&deleteResponse)
	assert.NoError(t, err, "No debería haber error al decodificar la respuesta")
	assert.Equal(t, "success", deleteResponse["status"], "Debería recibir status success")
	t.Log("✅ Eliminación de tarea exitosa")

	//  CASO 6: CREAR SEGUNDA TAREA PARA FILTROS
	t.Log("🧪 Probando caso 6: Crear segunda tarea para filtros")
	task2Payload := map[string]interface{}{
		"title":       "Segunda tarea para filtros",
		"description": "Tarea con estado completado",
		"due_date":    time.Now().Add(48 * time.Hour).Format(time.RFC3339),
		"priority":    "low",
		"status":      "complete",
		"assignee_id": assignee.ID,
	}
	task2Body, _ := json.Marshal(task2Payload)

	create2Req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(task2Body))
	create2Req.Header.Set("Content-Type", "application/json")

	create2Resp, err := app.Test(create2Req, -1)
	assert.NoError(t, err, "No debería haber error en la segunda creación")
	assert.Equal(t, http.StatusCreated, create2Resp.StatusCode, "Segunda creación debería ser exitosa")
	t.Log("✅ Segunda tarea creada exitosamente")

	//  CASO 7: FILTRAR TAREAS POR ESTADO CON HTTPTEST
	t.Log("🧪 Probando caso 7: Filtrar tareas por estado")
	filterReq := httptest.NewRequest(http.MethodGet, "/tasks?status=complete", nil)

	filterResp, err := app.Test(filterReq, -1)
	assert.NoError(t, err, "No debería haber error en el filtrado")
	assert.Equal(t, http.StatusOK, filterResp.StatusCode, "Filtrado debería ser exitoso")

	var filterResponse map[string]interface{}
	err = json.NewDecoder(filterResp.Body).Decode(&filterResponse)
	assert.NoError(t, err, "No debería haber error al decodificar la respuesta")
	assert.Equal(t, "success", filterResponse["status"], "Debería recibir status success")

	filteredTasks := filterResponse["data"].([]interface{})
	assert.Greater(t, len(filteredTasks), 0, "Debería haber tareas con estado completed")
	t.Logf("✅ Filtrado de tareas exitoso - %d tareas con estado 'completed'", len(filteredTasks))

	//  CASO 8: FILTRAR TAREAS POR PRIORIDAD
	t.Log("🧪 Probando caso 8: Filtrar tareas por prioridad")
	priorityReq := httptest.NewRequest(http.MethodGet, "/tasks?priority=low", nil)

	priorityResp, err := app.Test(priorityReq, -1)
	assert.NoError(t, err, "No debería haber error en el filtrado por prioridad")
	assert.Equal(t, http.StatusOK, priorityResp.StatusCode, "Filtrado por prioridad debería ser exitoso")
	t.Log("✅ Filtrado por prioridad exitoso")

	// Verificar conteo final de tareas en la transacción
	var finalCount int64
	tx.Model(&models.Task{}).Count(&finalCount)
	t.Logf("🔢 Total de tareas en la transacción: %d", finalCount)

	t.Log("🎉 Todos los casos de test completados exitosamente con httptest")
}

// Test adicional para verificar limpieza específica de las tareas
func TestTasksDatabaseCleanup(t *testing.T) {
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

	// Verificar que no hay tareas de test residuales en la base de datos real
	var taskCount int64
	tx.Model(&models.Task{}).Where("title LIKE ? OR description LIKE ?",
		"%prueba%", "%httptest%").Count(&taskCount)

	// Verificar que no hay usuarios de test residuales
	var userCount int64
	tx.Model(&models.User{}).Where("email LIKE ? OR email LIKE ?",
		"creator_%@example.com", "assignee_%@example.com").Count(&userCount)

	assert.Equal(t, int64(0), taskCount, "No debería haber tareas de test en la base de datos real")
	assert.Equal(t, int64(0), userCount, "No debería haber usuarios de test en la base de datos real")

	if taskCount == 0 && userCount == 0 {
		t.Log("✅ Base de datos limpia - no hay datos de test residuales")
	} else {
		if taskCount > 0 {
			t.Logf("⚠️  Encontradas %d tareas de test en la base de datos real", taskCount)

			// Mostrar tareas residuales
			var tasks []models.Task
			tx.Where("title LIKE ? OR description LIKE ?", "%prueba%", "%httptest%").Find(&tasks)
			for _, task := range tasks {
				t.Logf("   - Tarea residual: ID=%d, Título=%s, Estado=%s",
					task.ID, task.Title, task.Status)
			}
		}

		if userCount > 0 {
			t.Logf("⚠️  Encontrados %d usuarios de test en la base de datos real", userCount)

			// Mostrar usuarios residuales
			var users []models.User
			tx.Where("email LIKE ? OR email LIKE ?",
				"creator_%@example.com", "assignee_%@example.com").Find(&users)
			for _, user := range users {
				t.Logf("   - Usuario residual: ID=%v, Email=%s", user.ID, user.Email)
			}
		}
	}
}
