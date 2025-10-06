package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"microservices-platform/services/user-service/internal/database"
)

// UserRepository interface defines user data operations
type UserRepository interface {
	Create(ctx context.Context, user *database.User) error
	GetByID(ctx context.Context, id string) (*database.User, error)
	GetByEmail(ctx context.Context, email string) (*database.User, error)
	Update(ctx context.Context, user *database.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int, filter string) ([]*database.User, int64, error)
}

// userRepository implements UserRepository interface
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

// Create creates a new user
func (r *userRepository) Create(ctx context.Context, user *database.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id string) (*database.User, error) {
	var user database.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*database.User, error) {
	var user database.User
	err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *userRepository) Update(ctx context.Context, user *database.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete deletes a user
func (r *userRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&database.User{}, "id = ?", id).Error
}

// List lists users with pagination and filtering
func (r *userRepository) List(ctx context.Context, offset, limit int, filter string) ([]*database.User, int64, error) {
	var users []*database.User
	var total int64

	query := r.db.WithContext(ctx).Model(&database.User{})
	
	if filter != "" {
		query = query.Where("email ILIKE ? OR username ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
			"%"+filter+"%", "%"+filter+"%", "%"+filter+"%", "%"+filter+"%")
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}