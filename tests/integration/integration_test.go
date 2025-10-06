package integration

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userpb "microservices-platform/pkg/proto/user/v1"
	orderpb "microservices-platform/pkg/proto/order/v1"
	productpb "microservices-platform/pkg/proto/product/v1"
)

// TestSuite holds test configuration
type TestSuite struct {
	userClient         userpb.UserServiceClient
	orderClient        orderpb.OrderServiceClient
	productClient      productpb.ProductServiceClient
	userConn           *grpc.ClientConn
	orderConn          *grpc.ClientConn
	productConn        *grpc.ClientConn
}

// SetupTestSuite initializes the test suite
func SetupTestSuite(t *testing.T) *TestSuite {
	ctx := context.Background()

	// Connect to user service
	userConn, err := grpc.DialContext(ctx, "localhost:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to user service: %v", err)
	}

	// Connect to order service
	orderConn, err := grpc.DialContext(ctx, "localhost:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to order service: %v", err)
	}

	// Connect to product service
	productConn, err := grpc.DialContext(ctx, "localhost:8083", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to product service: %v", err)
	}

	return &TestSuite{
		userClient:    userpb.NewUserServiceClient(userConn),
		orderClient:   orderpb.NewOrderServiceClient(orderConn),
		productClient: productpb.NewProductServiceClient(productConn),
		userConn:      userConn,
		orderConn:     orderConn,
		productConn:   productConn,
	}
}

// TearDown cleans up test resources
func (ts *TestSuite) TearDown() {
	if ts.userConn != nil {
		ts.userConn.Close()
	}
	if ts.orderConn != nil {
		ts.orderConn.Close()
	}
	if ts.productConn != nil {
		ts.productConn.Close()
	}
}

// TestUserServiceIntegration tests user service integration
func TestUserServiceIntegration(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.TearDown()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test user creation
	createReq := &userpb.CreateUserRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	createResp, err := ts.userClient.CreateUser(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if createResp.User.Email != createReq.Email {
		t.Errorf("Expected email %s, got %s", createReq.Email, createResp.User.Email)
	}

	// Test user authentication
	authReq := &userpb.AuthenticateUserRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	authResp, err := ts.userClient.AuthenticateUser(ctx, authReq)
	if err != nil {
		t.Fatalf("Failed to authenticate user: %v", err)
	}

	if authResp.AccessToken == "" {
		t.Error("Expected access token, got empty string")
	}

	// Test user retrieval
	getReq := &userpb.GetUserRequest{
		UserId: createResp.User.UserId,
	}

	getResp, err := ts.userClient.GetUser(ctx, getReq)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if getResp.User.UserId != createResp.User.UserId {
		t.Errorf("Expected user ID %s, got %s", createResp.User.UserId, getResp.User.UserId)
	}
}

// TestOrderWorkflow tests the complete order workflow
func TestOrderWorkflow(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.TearDown()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First create a user
	userReq := &userpb.CreateUserRequest{
		Email:     "ordertest@example.com",
		Username:  "orderuser",
		Password:  "password123",
		FirstName: "Order",
		LastName:  "User",
	}

	userResp, err := ts.userClient.CreateUser(ctx, userReq)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a product
	productReq := &productpb.CreateProductRequest{
		Name:              "Test Product",
		Description:       "A test product",
		Price:             99.99,
		Category:          "Electronics",
		Brand:             "TestBrand",
		Sku:               "TEST-001",
		InventoryQuantity: 100,
	}

	productResp, err := ts.productClient.CreateProduct(ctx, productReq)
	if err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}

	// Create an order
	orderReq := &orderpb.CreateOrderRequest{
		UserId: userResp.User.UserId,
		Items: []*orderpb.CreateOrderItem{
			{
				ProductId: productResp.Product.ProductId,
				Quantity:  2,
			},
		},
		ShippingAddress: "123 Test St, Test City, TC 12345",
		BillingAddress:  "123 Test St, Test City, TC 12345",
	}

	orderResp, err := ts.orderClient.CreateOrder(ctx, orderReq)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	if orderResp.Order.UserId != userResp.User.UserId {
		t.Errorf("Expected user ID %s, got %s", userResp.User.UserId, orderResp.Order.UserId)
	}

	if len(orderResp.Order.Items) != 1 {
		t.Errorf("Expected 1 order item, got %d", len(orderResp.Order.Items))
	}

	// Update order status
	statusReq := &orderpb.UpdateOrderStatusRequest{
		OrderId: orderResp.Order.OrderId,
		Status:  orderpb.OrderStatus_ORDER_STATUS_CONFIRMED,
	}

	statusResp, err := ts.orderClient.UpdateOrderStatus(ctx, statusReq)
	if err != nil {
		t.Fatalf("Failed to update order status: %v", err)
	}

	if statusResp.Order.Status != orderpb.OrderStatus_ORDER_STATUS_CONFIRMED {
		t.Errorf("Expected order status %v, got %v", orderpb.OrderStatus_ORDER_STATUS_CONFIRMED, statusResp.Order.Status)
	}
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.TearDown()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	concurrency := 10
	errors := make(chan error, concurrency)
	results := make(chan *userpb.CreateUserResponse, concurrency)

	// Create multiple users concurrently
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			req := &userpb.CreateUserRequest{
				Email:     fmt.Sprintf("user%d@example.com", index),
				Username:  fmt.Sprintf("user%d", index),
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User",
			}

			resp, err := ts.userClient.CreateUser(ctx, req)
			if err != nil {
				errors <- err
				return
			}
			results <- resp
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < concurrency; i++ {
		select {
		case <-results:
			successCount++
		case err := <-errors:
			t.Logf("Error creating user: %v", err)
			errorCount++
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}

	if successCount < concurrency/2 {
		t.Errorf("Expected at least %d successful requests, got %d", concurrency/2, successCount)
	}

	t.Logf("Concurrent test results: %d successes, %d errors", successCount, errorCount)
}