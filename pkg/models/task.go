package models

import (
	"time"
)

// Task representa una tarea en el sistema
type Task struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Title       string    `json:"title" gorm:"not null"`
	Description string    `json:"description" gorm:"not null"`
	Status      string    `json:"status" gorm:"not null;default:'pending'"`
	Priority    string    `json:"priority" gorm:"not null;default:'medium'"`
	DueDate     time.Time `json:"due_date" gorm:"not null"`
	CreatorID   string    `json:"creator_id" gorm:"not null"` // ID del creador
	AssigneeID  string    `json:"assignee_id" gorm:"not null"`
	Creator     User      `json:"creator" gorm:"foreignKey:CreatorID"`
	Assignee    User      `json:"assignee" gorm:"foreignKey:AssigneeID"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskRequest representa la estructura para crear/actualizar una tarea
type TaskRequest struct {
	Title       string    `json:"title" validate:"required"`
	Description string    `json:"description" validate:"required"`
	Status      string    `json:"status" validate:"omitempty,oneof=pending in_progress complete"`
	Priority    string    `json:"priority" validate:"omitempty,oneof=low medium high"`
	DueDate     time.Time `json:"due_date" validate:"required"`
	AssigneeID  string    `json:"assignee_id"`
}

// TaskResponse representa la estructura de respuesta para una tarea
type TaskResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	DueDate     time.Time `json:"due_date"`
	CreatorID   string    `json:"creator_id"`
	AssigneeID  string    `json:"assignee_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
