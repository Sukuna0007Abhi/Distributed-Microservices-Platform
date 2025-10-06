package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"microservices-platform/services/order-service/internal/config"
	"microservices-platform/services/order-service/internal/database"
	"microservices-platform/services/order-service/internal/repository"
	productpb "microservices-platform/pkg/proto/product/v1"
	userpb "microservices-platform/pkg/proto/user/v1"
)

// OrderService interface defines order business logic operations
type OrderService interface {
	CreateOrder(ctx context.Context, userID string, items []CreateOrderItem, shippingAddress, billingAddress string) (*database.Order, error)
	GetOrder(ctx context.Context, id string) (*database.Order, error)
	UpdateOrderStatus(ctx context.Context, id, status string) (*database.Order, error)
	ListOrders(ctx context.Context, userID string, page, pageSize int, statusFilter string) ([]*database.Order, int64, error)
	CancelOrder(ctx context.Context, id, reason string) (*database.Order, error)
}

// CreateOrderItem represents an item to be added to an order
type CreateOrderItem struct {
	ProductID string
	Quantity  int32
}

// orderService implements OrderService interface
type orderService struct {
	orderRepo         repository.OrderRepository
	userServiceConn   *grpc.ClientConn
	productServiceConn *grpc.ClientConn
	userClient        userpb.UserServiceClient
	productClient     productpb.ProductServiceClient
}

// NewOrderService creates a new order service
func NewOrderService(orderRepo repository.OrderRepository, cfg *config.Config) OrderService {
	// Initialize gRPC connections
	userConn, err := grpc.Dial(cfg.UserServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to user service: %v", err)
		// In production, you might want to handle this more gracefully
	}

	productConn, err := grpc.Dial(cfg.ProductServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to product service: %v", err)
	}

	return &orderService{
		orderRepo:          orderRepo,
		userServiceConn:    userConn,
		productServiceConn: productConn,
		userClient:         userpb.NewUserServiceClient(userConn),
		productClient:      productpb.NewProductServiceClient(productConn),
	}
}

// CreateOrder creates a new order
func (s *orderService) CreateOrder(ctx context.Context, userID string, items []CreateOrderItem, shippingAddress, billingAddress string) (*database.Order, error) {
	// Verify user exists
	_, err := s.userClient.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to verify user: %v", err)
	}

	// Create order items and calculate total
	var orderItems []database.OrderItem
	var totalAmount float64

	for _, item := range items {
		// Get product details
		productResp, err := s.productClient.GetProduct(ctx, &productpb.GetProductRequest{ProductId: item.ProductID})
		if err != nil {
			return nil, fmt.Errorf("failed to get product %s: %v", item.ProductID, err)
		}

		product := productResp.Product
		unitPrice := product.Price
		totalPrice := unitPrice * float64(item.Quantity)

		orderItem := database.OrderItem{
			ProductID:   item.ProductID,
			ProductName: product.Name,
			Quantity:    item.Quantity,
			UnitPrice:   unitPrice,
			TotalPrice:  totalPrice,
		}

		orderItems = append(orderItems, orderItem)
		totalAmount += totalPrice
	}

	// Create order
	order := &database.Order{
		UserID:          userID,
		Items:           orderItems,
		TotalAmount:     totalAmount,
		Status:          "pending",
		ShippingAddress: shippingAddress,
		BillingAddress:  billingAddress,
	}

	err = s.orderRepo.Create(ctx, order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// GetOrder retrieves an order by ID
func (s *orderService) GetOrder(ctx context.Context, id string) (*database.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("order not found")
	}

	return order, nil
}

// UpdateOrderStatus updates the status of an order
func (s *orderService) UpdateOrderStatus(ctx context.Context, id, status string) (*database.Order, error) {
	// Verify order exists
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("order not found")
	}

	// Update status
	err = s.orderRepo.UpdateStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}

	// Return updated order
	return s.orderRepo.GetByID(ctx, id)
}

// ListOrders lists orders for a user with pagination and filtering
func (s *orderService) ListOrders(ctx context.Context, userID string, page, pageSize int, statusFilter string) ([]*database.Order, int64, error) {
	offset := (page - 1) * pageSize
	return s.orderRepo.ListByUserID(ctx, userID, offset, pageSize, statusFilter)
}

// CancelOrder cancels an order
func (s *orderService) CancelOrder(ctx context.Context, id, reason string) (*database.Order, error) {
	// Get current order
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("order not found")
	}

	// Check if order can be cancelled
	if order.Status == "shipped" || order.Status == "delivered" || order.Status == "cancelled" {
		return nil, fmt.Errorf("cannot cancel order with status: %s", order.Status)
	}

	// Update status to cancelled
	err = s.orderRepo.UpdateStatus(ctx, id, "cancelled")
	if err != nil {
		return nil, err
	}

	// Return updated order
	return s.orderRepo.GetByID(ctx, id)
}