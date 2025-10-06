package database

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewConnection creates a new database connection
func NewConnection(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate models
	err = db.AutoMigrate(&Product{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Product model
type Product struct {
	ID                string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name              string         `gorm:"not null;index"`
	Description       string         `gorm:"type:text"`
	Price             float64        `gorm:"not null;index"`
	Category          string         `gorm:"not null;index"`
	Brand             string         `gorm:"not null;index"`
	SKU               string         `gorm:"unique;not null"`
	InventoryQuantity int32          `gorm:"not null;default:0"`
	Images            []string `gorm:"type:text[]"`
	Status            string         `gorm:"default:active;index"`
	CreatedAt         time.Time      `gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime"`
}

// InventoryLog model for tracking inventory changes
type InventoryLog struct {
	ID            string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ProductID     string    `gorm:"not null;index"`
	QuantityChange int32    `gorm:"not null"`
	Reason        string    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}