# API Documentation

## Overview

This document describes the REST API endpoints exposed by the API Gateway for the distributed microservices platform.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Most endpoints require authentication using JWT tokens. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

## User Service Endpoints

### Create User
- **POST** `/users`
- **Description**: Create a new user account
- **Request Body**:
```json
{
  "email": "user@example.com",
  "username": "username",
  "password": "password123",
  "first_name": "John",
  "last_name": "Doe"
}
```
- **Response**:
```json
{
  "user": {
    "user_id": "uuid",
    "email": "user@example.com",
    "username": "username",
    "first_name": "John",
    "last_name": "Doe",
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

### Authenticate User
- **POST** `/auth/login`
- **Description**: Authenticate user and get access token
- **Request Body**:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```
- **Response**:
```json
{
  "access_token": "jwt-token",
  "refresh_token": "refresh-token",
  "user": {...},
  "expires_in": 86400
}
```

### Get User
- **GET** `/users/{id}`
- **Description**: Get user by ID
- **Headers**: `Authorization: Bearer <token>`
- **Response**: User object

### Update User
- **PUT** `/users/{id}`
- **Description**: Update user information
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**: Partial user object with fields to update

### Delete User
- **DELETE** `/users/{id}`
- **Description**: Delete user account
- **Headers**: `Authorization: Bearer <token>`

### List Users
- **GET** `/users?page=1&page_size=20&filter=search`
- **Description**: List users with pagination and filtering
- **Headers**: `Authorization: Bearer <token>`
- **Query Parameters**:
  - `page`: Page number (default: 1)
  - `page_size`: Items per page (default: 20)
  - `filter`: Search filter

## Product Service Endpoints

### Create Product
- **POST** `/products`
- **Description**: Create a new product
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "name": "Product Name",
  "description": "Product description",
  "price": 99.99,
  "category": "Electronics",
  "brand": "BrandName",
  "sku": "PROD-001",
  "inventory_quantity": 100,
  "images": ["http://example.com/image1.jpg"]
}
```

### Get Product
- **GET** `/products/{id}`
- **Description**: Get product by ID

### Update Product
- **PUT** `/products/{id}`
- **Description**: Update product information
- **Headers**: `Authorization: Bearer <token>`

### Delete Product
- **DELETE** `/products/{id}`
- **Description**: Delete product
- **Headers**: `Authorization: Bearer <token>`

### List Products
- **GET** `/products?page=1&page_size=20&category=Electronics&brand=BrandName`
- **Description**: List products with filtering

### Search Products
- **GET** `/products/search?query=search+term&page=1&page_size=20`
- **Description**: Search products by query

### Update Inventory
- **PUT** `/products/{id}/inventory`
- **Description**: Update product inventory
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "quantity_change": -5,
  "reason": "Sale"
}
```

## Order Service Endpoints

### Create Order
- **POST** `/orders`
- **Description**: Create a new order
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "user_id": "user-uuid",
  "items": [
    {
      "product_id": "product-uuid",
      "quantity": 2
    }
  ],
  "shipping_address": "123 Main St, City, State 12345",
  "billing_address": "123 Main St, City, State 12345"
}
```

### Get Order
- **GET** `/orders/{id}`
- **Description**: Get order by ID
- **Headers**: `Authorization: Bearer <token>`

### Update Order Status
- **PUT** `/orders/{id}/status`
- **Description**: Update order status
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "status": "confirmed"
}
```

### Cancel Order
- **POST** `/orders/{id}/cancel`
- **Description**: Cancel an order
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "reason": "Customer request"
}
```

### List Orders
- **GET** `/orders?user_id=uuid&status=pending&page=1&page_size=20`
- **Description**: List orders with filtering
- **Headers**: `Authorization: Bearer <token>`

## Payment Service Endpoints

### Process Payment
- **POST** `/payments`
- **Description**: Process a payment
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "order_id": "order-uuid",
  "user_id": "user-uuid",
  "amount": 199.98,
  "currency": "USD",
  "method": "credit_card",
  "card_details": {
    "card_number": "4111111111111111",
    "expiry_month": "12",
    "expiry_year": "2025",
    "cvv": "123",
    "holder_name": "John Doe"
  }
}
```

### Get Payment
- **GET** `/payments/{id}`
- **Description**: Get payment by ID
- **Headers**: `Authorization: Bearer <token>`

### Refund Payment
- **POST** `/payments/{id}/refund`
- **Description**: Refund a payment
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "amount": 99.99,
  "reason": "Product return"
}
```

### List Payments
- **GET** `/payments?user_id=uuid&order_id=uuid&status=completed`
- **Description**: List payments with filtering
- **Headers**: `Authorization: Bearer <token>`

### Payment Webhook
- **POST** `/payments/webhook`
- **Description**: Handle payment gateway webhooks
- **Request Body**: Gateway-specific payload

## Notification Service Endpoints

### Send Notification
- **POST** `/notifications`
- **Description**: Send a notification
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "user_id": "user-uuid",
  "title": "Order Shipped",
  "message": "Your order has been shipped!",
  "type": "order_shipped",
  "channels": ["email", "push"],
  "immediate": true
}
```

### Get Notification
- **GET** `/notifications/{id}`
- **Description**: Get notification by ID
- **Headers**: `Authorization: Bearer <token>`

### List Notifications
- **GET** `/notifications?user_id=uuid&type=order_shipped&unread_only=true`
- **Description**: List notifications for a user
- **Headers**: `Authorization: Bearer <token>`

### Mark as Read
- **PUT** `/notifications/{id}/read`
- **Description**: Mark notification as read
- **Headers**: `Authorization: Bearer <token>`

### Delete Notification
- **DELETE** `/notifications/{id}`
- **Description**: Delete a notification
- **Headers**: `Authorization: Bearer <token>`

### Subscribe to Notifications
- **POST** `/notifications/subscribe`
- **Description**: Subscribe to notification types
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
```json
{
  "user_id": "user-uuid",
  "types": ["order_confirmation", "order_shipped"],
  "channels": ["email", "push"],
  "preferences": {
    "email_frequency": "immediate",
    "push_enabled": "true"
  }
}
```

## Error Responses

All endpoints may return these error responses:

### 400 Bad Request
```json
{
  "error": "invalid_request",
  "message": "The request is invalid or malformed"
}
```

### 401 Unauthorized
```json
{
  "error": "unauthorized",
  "message": "Authentication required"
}
```

### 403 Forbidden
```json
{
  "error": "forbidden",
  "message": "Access denied"
}
```

### 404 Not Found
```json
{
  "error": "not_found",
  "message": "Resource not found"
}
```

### 500 Internal Server Error
```json
{
  "error": "internal_error",
  "message": "An internal server error occurred"
}
```

## Rate Limiting

API requests are rate limited to:
- 100 requests per minute per IP address
- 1000 requests per hour per authenticated user

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Request limit per time window
- `X-RateLimit-Remaining`: Requests remaining in current window
- `X-RateLimit-Reset`: Time when the rate limit resets

## Status Codes Reference

- **200 OK**: Request successful
- **201 Created**: Resource created successfully
- **204 No Content**: Request successful, no content returned
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Access denied
- **404 Not Found**: Resource not found
- **409 Conflict**: Resource conflict
- **422 Unprocessable Entity**: Validation failed
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Server error