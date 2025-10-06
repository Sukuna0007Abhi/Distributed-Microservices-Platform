package database

import (
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
	err = db.AutoMigrate(&User{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// User model
type User struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email     string `gorm:"unique;not null"`
	Username  string `gorm:"unique;not null"`
	Password  string `gorm:"not null"`
	FirstName string
	LastName  string
	Status    string `gorm:"default:active"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	UpdatedAt int64  `gorm:"autoUpdateTime"`
}