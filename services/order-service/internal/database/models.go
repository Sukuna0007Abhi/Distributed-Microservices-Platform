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
	err = db.AutoMigrate(&Order{}, &OrderItem{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Order model
type Order struct {
	ID              string      `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID          string      `gorm:"not null;index"`
	Items           []OrderItem `gorm:"foreignKey:OrderID"`
	TotalAmount     float64     `gorm:"not null"`
	Status          string      `gorm:"default:pending"`
	ShippingAddress string      `gorm:"not null"`
	BillingAddress  string      `gorm:"not null"`
	CreatedAt       time.Time   `gorm:"autoCreateTime"`
	UpdatedAt       time.Time   `gorm:"autoUpdateTime"`
}

// OrderItem model
type OrderItem struct {
	ID          string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	OrderID     string  `gorm:"not null;index"`
	ProductID   string  `gorm:"not null"`
	ProductName string  `gorm:"not null"`
	Quantity    int32   `gorm:"not null"`
	UnitPrice   float64 `gorm:"not null"`
	TotalPrice  float64 `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}