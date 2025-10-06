package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"microservices-platform/services/order-service/internal/database"
)

// OrderRepository interface defines order data operations
type OrderRepository interface {
	Create(ctx context.Context, order *database.Order) error
	GetByID(ctx context.Context, id string) (*database.Order, error)
	Update(ctx context.Context, order *database.Order) error
	Delete(ctx context.Context, id string) error
	ListByUserID(ctx context.Context, userID string, offset, limit int, statusFilter string) ([]*database.Order, int64, error)
	UpdateStatus(ctx context.Context, id, status string) error
}

// orderRepository implements OrderRepository interface
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{
		db: db,
	}
}

// Create creates a new order
func (r *orderRepository) Create(ctx context.Context, order *database.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// GetByID retrieves an order by ID
func (r *orderRepository) GetByID(ctx context.Context, id string) (*database.Order, error) {
	var order database.Order
	err := r.db.WithContext(ctx).Preload("Items").First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// Update updates an order
func (r *orderRepository) Update(ctx context.Context, order *database.Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}

// Delete deletes an order
func (r *orderRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&database.Order{}, "id = ?", id).Error
}

// ListByUserID lists orders for a specific user with pagination and filtering
func (r *orderRepository) ListByUserID(ctx context.Context, userID string, offset, limit int, statusFilter string) ([]*database.Order, int64, error) {
	var orders []*database.Order
	var total int64

	query := r.db.WithContext(ctx).Model(&database.Order{}).Where("user_id = ?", userID)
	
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results with items
	if err := query.Preload("Items").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// UpdateStatus updates the status of an order
func (r *orderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).Model(&database.Order{}).Where("id = ?", id).Update("status", status).Error
}