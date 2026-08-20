package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Workflow struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UserID      string    `gorm:"type:uuid;index" json:"user_id"`
	Status      string    `gorm:"default:pending" json:"status"`
	Config      JSON      `gorm:"type:jsonb" json:"config"`
	Result      JSON      `gorm:"type:jsonb" json:"result"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (w *Workflow) BeforeCreate(tx *gorm.DB) error {
	w.ID = uuid.New().String()
	return nil
}

// JSON type for PostgreSQL JSONB
type JSON map[string]interface{}