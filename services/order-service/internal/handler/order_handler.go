package handler

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"microservices-platform/services/order-service/internal/service"
	pb "microservices-platform/pkg/proto/order/v1"
)

// OrderHandler implements the gRPC OrderService
type OrderHandler struct {
	pb.UnimplementedOrderServiceServer
	orderService service.OrderService
	tracer       trace.Tracer
}

// NewOrderHandler creates a new OrderHandler
func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		tracer:       otel.Tracer("order-service"),
	}
}

// CreateOrder creates a new order
func (h *OrderHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	ctx, span := h.tracer.Start(ctx, "OrderHandler.CreateOrder")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.user_id", req.UserId),
		attribute.Int("order.items_count", len(req.Items)),
	)

	// Convert request items to service items
	var items []service.CreateOrderItem
	for _, item := range req.Items {
		items = append(items, service.CreateOrderItem{
			ProductID: item.ProductId,
			Quantity:  item.Quantity,
		})
	}

	order, err := h.orderService.CreateOrder(ctx, req.UserId, items, req.ShippingAddress, req.BillingAddress)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}

	return &pb.CreateOrderResponse{
		Order: h.convertToProtoOrder(order),
	}, nil
}

// GetOrder retrieves an order by ID
func (h *OrderHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	ctx, span := h.tracer.Start(ctx, "OrderHandler.GetOrder")
	defer span.End()

	span.SetAttributes(attribute.String("order.id", req.OrderId))

	order, err := h.orderService.GetOrder(ctx, req.OrderId)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.NotFound, "order not found: %v", err)
	}

	return &pb.GetOrderResponse{
		Order: h.convertToProtoOrder(order),
	}, nil
}

// UpdateOrderStatus updates an order status
func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.UpdateOrderStatusResponse, error) {
	ctx, span := h.tracer.Start(ctx, "OrderHandler.UpdateOrderStatus")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.id", req.OrderId),
		attribute.String("order.status", req.Status.String()),
	)

	statusStr := h.convertOrderStatusToString(req.Status)
	order, err := h.orderService.UpdateOrderStatus(ctx, req.OrderId, statusStr)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to update order status: %v", err)
	}

	return &pb.UpdateOrderStatusResponse{
		Order: h.convertToProtoOrder(order),
	}, nil
}

// ListOrders lists orders for a user
func (h *OrderHandler) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	ctx, span := h.tracer.Start(ctx, "OrderHandler.ListOrders")
	defer span.End()

	span.SetAttributes(
		attribute.String("orders.user_id", req.UserId),
		attribute.Int64("pagination.page", int64(req.Page)),
		attribute.Int64("pagination.page_size", int64(req.PageSize)),
	)

	statusFilter := ""
	if req.StatusFilter != pb.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		statusFilter = h.convertOrderStatusToString(req.StatusFilter)
	}

	orders, total, err := h.orderService.ListOrders(ctx, req.UserId, int(req.Page), int(req.PageSize), statusFilter)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to list orders: %v", err)
	}

	var protoOrders []*pb.Order
	for _, order := range orders {
		protoOrders = append(protoOrders, h.convertToProtoOrder(order))
	}

	return &pb.ListOrdersResponse{
		Orders:     protoOrders,
		TotalCount: int32(total),
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// CancelOrder cancels an order
func (h *OrderHandler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	ctx, span := h.tracer.Start(ctx, "OrderHandler.CancelOrder")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.id", req.OrderId),
		attribute.String("cancel.reason", req.Reason),
	)

	order, err := h.orderService.CancelOrder(ctx, req.OrderId, req.Reason)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to cancel order: %v", err)
	}

	return &pb.CancelOrderResponse{
		Order: h.convertToProtoOrder(order),
	}, nil
}

// convertToProtoOrder converts database order to protobuf order
func (h *OrderHandler) convertToProtoOrder(order *service.Order) *pb.Order {
	var items []*pb.OrderItem
	for _, item := range order.Items {
		items = append(items, &pb.OrderItem{
			ItemId:      item.ID,
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			TotalPrice:  item.TotalPrice,
		})
	}

	return &pb.Order{
		OrderId:         order.ID,
		UserId:          order.UserID,
		Items:           items,
		TotalAmount:     order.TotalAmount,
		Status:          h.convertStringToOrderStatus(order.Status),
		ShippingAddress: order.ShippingAddress,
		BillingAddress:  order.BillingAddress,
		CreatedAt:       timestamppb.New(order.CreatedAt),
		UpdatedAt:       timestamppb.New(order.UpdatedAt),
	}
}

// convertOrderStatusToString converts protobuf order status to string
func (h *OrderHandler) convertOrderStatusToString(status pb.OrderStatus) string {
	switch status {
	case pb.OrderStatus_ORDER_STATUS_PENDING:
		return "pending"
	case pb.OrderStatus_ORDER_STATUS_CONFIRMED:
		return "confirmed"
	case pb.OrderStatus_ORDER_STATUS_PROCESSING:
		return "processing"
	case pb.OrderStatus_ORDER_STATUS_SHIPPED:
		return "shipped"
	case pb.OrderStatus_ORDER_STATUS_DELIVERED:
		return "delivered"
	case pb.OrderStatus_ORDER_STATUS_CANCELLED:
		return "cancelled"
	case pb.OrderStatus_ORDER_STATUS_REFUNDED:
		return "refunded"
	default:
		return "pending"
	}
}

// convertStringToOrderStatus converts string to protobuf order status
func (h *OrderHandler) convertStringToOrderStatus(status string) pb.OrderStatus {
	switch status {
	case "pending":
		return pb.OrderStatus_ORDER_STATUS_PENDING
	case "confirmed":
		return pb.OrderStatus_ORDER_STATUS_CONFIRMED
	case "processing":
		return pb.OrderStatus_ORDER_STATUS_PROCESSING
	case "shipped":
		return pb.OrderStatus_ORDER_STATUS_SHIPPED
	case "delivered":
		return pb.OrderStatus_ORDER_STATUS_DELIVERED
	case "cancelled":
		return pb.OrderStatus_ORDER_STATUS_CANCELLED
	case "refunded":
		return pb.OrderStatus_ORDER_STATUS_REFUNDED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}