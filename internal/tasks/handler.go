package tasks

import (
	"fmt"
	"legendaryum/internal/config"
	"legendaryum/pkg/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Handler maneja las operaciones relacionadas con tareas
type Handler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewHandler crea una nueva instancia del handler de tareas
func NewHandler(db *gorm.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:  db,
		cfg: cfg,
	}
}

// Create godoc
// @Summary Crear una nueva tarea
// @Description Crea una nueva tarea en el sistema Legendaryum. El creador se toma del token JWT. Si assignee_id no se especifica, la tarea se asigna al creador.
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body models.TaskRequest true "Datos necesarios para crear una tarea"
// @Security Bearer
// @Success 201 {object} models.Task "Tarea creada exitosamente" // Usar models.Task para la respuesta completa
// @Failure 400 {object} models.ErrorResponse "Error en los datos de entrada (JSON inválido, campos requeridos, assignee no encontrado)"
// @Failure 401 {object} models.ErrorResponse "No autorizado (token JWT faltante o inválido)"
// @Failure 500 {object} models.ErrorResponse "Error interno del servidor"
// @Router /tasks [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var req models.TaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Error al procesar la solicitud: JSON inválido.",
		})
	}

	// Validar campos requeridos (ya existen, se pueden mejorar los mensajes)
	if req.Title == "" || req.Description == "" || req.DueDate.IsZero() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Los campos 'title', 'description' y 'due_date' son requeridos.",
		})
	}

	// Obtener el ID del usuario del token JWT
	creatorID, ok := c.Locals("user_id").(string)
	if !ok || creatorID == "" {
		// Esto no debería ocurrir si el middleware AuthMiddleware funciona, pero es una seguridad
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Usuario no autenticado.",
		})
	}

	// Si no se especifica assignee_id, usar el ID del creador
	assigneeID := req.AssigneeID
	if assigneeID == "" {
		assigneeID = creatorID
	}

	// Validar que el assignee existe si se especificó uno diferente al creador
	if assigneeID != creatorID {
		var assignee models.User
		if err := h.db.First(&assignee, "id = ?", assigneeID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": fmt.Sprintf("El usuario asignado con ID %s no existe.", assigneeID),
				})
			}
			// Loggear error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error interno al verificar usuario asignado.",
			})
		}
	}

	// Establecer valores por defecto si no se especifican (basado en tags validate omitempty)
	status := req.Status
	if status == "" {
		status = "pending"
	}
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	task := models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      status,   // Usar el valor (por defecto o del request)
		Priority:    priority, // Usar el valor (por defecto o del request)
		DueDate:     req.DueDate,
		CreatorID:   creatorID,
		AssigneeID:  assigneeID,
	}

	if err := h.db.Create(&task).Error; err != nil {
		// Loggear error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error interno al crear la tarea.",
		})
	}

	// Cargar las relaciones creator y assignee para la respuesta
	if err := h.db.Preload("Creator").Preload("Assignee").First(&task, task.ID).Error; err != nil {
		// Loggear error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error interno al cargar los datos de la tarea creada.",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Tarea creada exitosamente.",
		"data":    task,
	})
}

// List godoc
// @Summary Listar tareas
// @Description Obtiene todas las tareas donde el usuario autenticado es el creador o el asignado. Permite filtrar por estado y prioridad.
// @Tags tasks
// @Accept json
// @Produce json
// @Param status query string false "Filtrar por estado de la tarea ('pending', 'in_progress', 'complete')" Enums: pending, in_progress, complete
// @Param priority query string false "Filtrar por prioridad de la tarea ('low', 'medium', 'high')" Enums: low, medium, high
// @Security Bearer
// @Success 200 {array} models.Task "Lista de tareas" // Usar array de models.Task
// @Failure 401 {object} models.ErrorResponse "No autorizado (token JWT faltante o inválido)"
// @Failure 500 {object} models.ErrorResponse "Error interno del servidor"
// @Router /tasks [get]
func (h *Handler) List(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Usuario no autenticado.",
		})
	}

	// Obtener filtros de query params
	status := c.Query("status")
	priority := c.Query("priority")

	// Construir la consulta base
	query := h.db.Where("creator_id = ? OR assignee_id = ?", userID, userID)

	// Aplicar filtros si existen
	if status != "" {
		// Opcional: validar que el estado es un valor válido del ENUM si es necesario
		query = query.Where("status = ?", status)
	}
	if priority != "" {
		// Opcional: validar que la prioridad es un valor válido del ENUM si es necesario
		query = query.Where("priority = ?", priority)
	}

	var tasks []models.Task
	if err := query.Preload("Creator").Preload("Assignee").Find(&tasks).Error; err != nil {
		// Loggear error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error interno al obtener las tareas.",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Tareas obtenidas exitosamente.",
		"data":    tasks,
	})
}

// Get godoc
// @Summary Obtener una tarea específica
// @Description Obtiene los detalles de una tarea por su ID si el usuario autenticado es el creador o el asignado.
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "ID numérico de la tarea a obtener" Format(uint)
// @Security Bearer
// @Success 200 {object} models.Task "Detalles de la tarea" // Usar models.Task
// @Failure 400 {object} models.ErrorResponse "ID inválido"
// @Failure 401 {object} models.ErrorResponse "No autorizado (token JWT faltante o inválido)"
// @Failure 403 {object} models.ErrorResponse "Acceso denegado (la tarea no pertenece al usuario)"
// @Failure 404 {object} models.ErrorResponse "Tarea no encontrada"
// @Failure 500 {object} models.ErrorResponse "Error interno del servidor"
// @Router /tasks/{id} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID de tarea inválido. Debe ser un número entero.",
		})
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Usuario no autenticado.",
		})
	}

	var task models.Task
	if err := h.db.Where("id = ? AND (creator_id = ? OR assignee_id = ?)", taskID, userID, userID).
		Preload("Creator").Preload("Assignee").
		First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Si no se encuentra O si no pertenece al usuario, retorna 404
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Tarea no encontrada o no tienes permiso para verla.",
			})
		}
		// Loggear error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error interno al obtener la tarea.",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Tarea obtenida exitosamente.",
		"data":    task,
	})
}

// Update godoc
// @Summary Actualizar una tarea
// @Description Actualiza los datos de una tarea existente. Solo el creador de la tarea puede actualizarla.
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "ID numérico de la tarea a actualizar" Format(uint)
// @Param request body models.TaskRequest true "Datos para actualizar la tarea"
// @Security Bearer
// @Success 200 {object} models.Task "Tarea actualizada exitosamente" // Usar models.Task
// @Failure 400 {object} models.ErrorResponse "Error en los datos de entrada (ID inválido, JSON inválido, nuevo assignee no encontrado)"
// @Failure 401 {object} models.ErrorResponse "No autorizado (token JWT faltante o inválido)"
// @Failure 403 {object} models.ErrorResponse "Permiso denegado (solo el creador puede actualizar)"
// @Failure 404 {object} models.ErrorResponse "Tarea no encontrada"
// @Failure 500 {object} models.ErrorResponse "Error interno del servidor"
// @Router /tasks/{id} [put]
func (h *Handler) Update(c *fiber.Ctx) error {
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID de tarea inválido. Debe ser un número entero.",
		})
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Usuario no autenticado.",
		})
	}

	var req models.TaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Error al procesar la solicitud: JSON inválido.",
		})
	}

	var task models.Task
	// Buscar tarea y verificar que el usuario es el creador
	if err := h.db.Where("id = ? AND creator_id = ?", taskID, userID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Si no se encuentra O si no es el creador, retorna 403 (Permiso denegado)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "No tienes permiso para actualizar esta tarea.",
			})
		}
		// Loggear error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error interno al obtener la tarea para actualizar.",
		})
	}

	// Aplicar actualizaciones desde la solicitud si los campos están presentes
	// Fiber BodyParser con structs y validate tags ayuda, pero GORM Save necesita que copies
	// los campos que realmente quieres actualizar.
	updates := make(map[string]interface{})

	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	// Comprobar si DueDate no es la hora cero por defecto
	if !req.DueDate.IsZero() {
		updates["due_date"] = req.DueDate
	}
	// Comprobar si Status está en la lista de valores permitidos o no está vacío
	if req.Status != "" {
		// Opcional: Validar req.Status con los valores del ENUM antes de añadir a updates
		updates["status"] = req.Status
	}
	// Comprobar si Priority está en la lista de valores permitidos o no está vacío
	if req.Priority != "" {
		// Opcional: Validar req.Priority con los valores del ENUM antes de añadir a updates
		updates["priority"] = req.Priority
	}
	if req.AssigneeID != "" {
		// Validar que el nuevo assignee existe
		var newAssignee models.User
		if err := h.db.First(&newAssignee, "id = ?", req.AssigneeID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": fmt.Sprintf("El nuevo usuario asignado con ID %s no existe.", req.AssigneeID),
				})
			}
			// Loggear error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error interno al verificar nuevo usuario asignado.",
			})
		}
		updates["assignee_id"] = req.AssigneeID
	}

	// Usar Updates para actualizar solo los campos proporcionados
	if len(updates) > 0 {
		if err := h.db.Model(&task).Updates(updates).Error; err != nil {
			// Loggear error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Error interno al actualizar la tarea.",
			})
		}
	} else {
		// Si no hay campos para actualizar, retornar éxito con la tarea actual
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "No se proporcionaron campos para actualizar.",
			"data":    task, // Devolver la tarea sin modificar si no hay updates
		})
	}

	// Cargar las relaciones creator y assignee después de actualizar
	if err := h.db.Preload("Creator").Preload("Assignee").First(&task, taskID).Error; err != nil {
		// Loggear error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error interno al cargar los datos actualizados de la tarea.",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Tarea actualizada exitosamente.",
		"data":    task,
	})
}

// Delete godoc
// @Summary Eliminar una tarea
// @Description Elimina una tarea existente por su ID. Solo el creador de la tarea puede eliminarla.
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "ID numérico de la tarea a eliminar" Format(uint)
// @Security Bearer
// @Success 200 {object} models.SuccessResponse "Tarea eliminada exitosamente" // Usar una respuesta simple para eliminación
// @Failure 400 {object} models.ErrorResponse "ID inválido"
// @Failure 401 {object} models.ErrorResponse "No autorizado (token JWT faltante o inválido)"
// @Failure 403 {object} models.ErrorResponse "Permiso denegado (solo el creador puede eliminar)"
// @Failure 404 {object} models.ErrorResponse "Tarea no encontrada"
// @Failure 500 {object} models.ErrorResponse "Error interno del servidor"
// @Router /tasks/{id} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID de tarea inválido. Debe ser un número entero.",
		})
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Usuario no autenticado.",
		})
	}

	var task models.Task
	// Buscar tarea y verificar que el usuario es el creador
	if err := h.db.Where("id = ? AND creator_id = ?", taskID, userID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Si no se encuentra O si no es el creador, retorna 403
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "No tienes permiso para eliminar esta tarea.",
			})
		}
		// Loggear error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error interno al obtener la tarea para eliminar.",
		})
	}

	// Eliminar la tarea
	if err := h.db.Delete(&task).Error; err != nil {
		// Loggear error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error interno al eliminar la tarea.",
		})
	}

	// Definir una estructura simple para la respuesta de éxito si no hay datos que devolver
	// Puede ser un mapa o una estructura models.SuccessResponse simple
	// type SuccessResponse struct { Status string `json:"status" example:"success"` Message string `json:"message" example:"Operación exitosa"` }
	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Tarea eliminada exitosamente.",
		"data":    nil, // Omitir 'data' o poner nil si no se devuelve cuerpo
	})
}
