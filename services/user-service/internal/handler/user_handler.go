package handler

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"microservices-platform/services/user-service/internal/service"
	pb "microservices-platform/pkg/proto/user/v1"
)

// UserHandler implements the gRPC UserService
type UserHandler struct {
	pb.UnimplementedUserServiceServer
	userService service.UserService
	tracer      trace.Tracer
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		tracer:      otel.Tracer("user-service"),
	}
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.CreateUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("user.username", req.Username),
	)

	user, err := h.userService.CreateUser(ctx, req.Email, req.Username, req.Password, req.FirstName, req.LastName)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &pb.CreateUserResponse{
		User: h.convertToProtoUser(user),
	}, nil
}

// GetUser retrieves a user by ID
func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.GetUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.UserId))

	user, err := h.userService.GetUser(ctx, req.UserId)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	return &pb.GetUserResponse{
		User: h.convertToProtoUser(user),
	}, nil
}

// UpdateUser updates a user
func (h *UserHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.UpdateUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.UserId),
		attribute.String("user.email", req.Email),
	)

	statusStr := ""
	switch req.Status {
	case pb.UserStatus_USER_STATUS_ACTIVE:
		statusStr = "active"
	case pb.UserStatus_USER_STATUS_INACTIVE:
		statusStr = "inactive"
	case pb.UserStatus_USER_STATUS_SUSPENDED:
		statusStr = "suspended"
	}

	user, err := h.userService.UpdateUser(ctx, req.UserId, req.Email, req.Username, req.FirstName, req.LastName, statusStr)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	return &pb.UpdateUserResponse{
		User: h.convertToProtoUser(user),
	}, nil
}

// DeleteUser deletes a user
func (h *UserHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.DeleteUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.UserId))

	err := h.userService.DeleteUser(ctx, req.UserId)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	return &pb.DeleteUserResponse{
		Success: true,
	}, nil
}

// ListUsers lists users with pagination
func (h *UserHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.ListUsers")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("pagination.page", int64(req.Page)),
		attribute.Int64("pagination.page_size", int64(req.PageSize)),
	)

	users, total, err := h.userService.ListUsers(ctx, int(req.Page), int(req.PageSize), req.Filter)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	var protoUsers []*pb.User
	for _, user := range users {
		protoUsers = append(protoUsers, h.convertToProtoUser(user))
	}

	return &pb.ListUsersResponse{
		Users:      protoUsers,
		TotalCount: int32(total),
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// AuthenticateUser authenticates a user
func (h *UserHandler) AuthenticateUser(ctx context.Context, req *pb.AuthenticateUserRequest) (*pb.AuthenticateUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.AuthenticateUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", req.Email))

	user, token, err := h.userService.AuthenticateUser(ctx, req.Email, req.Password)
	if err != nil {
		span.RecordError(err)
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	return &pb.AuthenticateUserResponse{
		AccessToken: token,
		User:        h.convertToProtoUser(user),
		ExpiresIn:   86400, // 24 hours
	}, nil
}

// convertToProtoUser converts database user to protobuf user
func (h *UserHandler) convertToProtoUser(user *service.User) *pb.User {
	var status pb.UserStatus
	switch user.Status {
	case "active":
		status = pb.UserStatus_USER_STATUS_ACTIVE
	case "inactive":
		status = pb.UserStatus_USER_STATUS_INACTIVE
	case "suspended":
		status = pb.UserStatus_USER_STATUS_SUSPENDED
	default:
		status = pb.UserStatus_USER_STATUS_UNSPECIFIED
	}

	return &pb.User{
		UserId:    user.ID,
		Email:     user.Email,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Status:    status,
		CreatedAt: timestamppb.New(time.Unix(user.CreatedAt, 0)),
		UpdatedAt: timestamppb.New(time.Unix(user.UpdatedAt, 0)),
	}
}