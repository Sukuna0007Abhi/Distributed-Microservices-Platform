package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
	"microservices-platform/services/user-service/internal/database"
	"microservices-platform/services/user-service/internal/repository"
)

// UserService interface defines user business logic operations
type UserService interface {
	CreateUser(ctx context.Context, email, username, password, firstName, lastName string) (*database.User, error)
	GetUser(ctx context.Context, id string) (*database.User, error)
	UpdateUser(ctx context.Context, id, email, username, firstName, lastName, status string) (*database.User, error)
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, page, pageSize int, filter string) ([]*database.User, int64, error)
	AuthenticateUser(ctx context.Context, email, password string) (*database.User, string, error)
}

// userService implements UserService interface
type userService struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

// NewUserService creates a new user service
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{
		userRepo:  userRepo,
		jwtSecret: "your-secret-key", // In production, this should come from config
	}
}

// CreateUser creates a new user with hashed password
func (s *userService) CreateUser(ctx context.Context, email, username, password, firstName, lastName string) (*database.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	user := &database.User{
		Email:     email,
		Username:  username,
		Password:  hashedPassword,
		FirstName: firstName,
		LastName:  lastName,
		Status:    "active",
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	// Don't return password hash
	user.Password = ""
	return user, nil
}

// GetUser retrieves a user by ID
func (s *userService) GetUser(ctx context.Context, id string) (*database.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Don't return password hash
	user.Password = ""
	return user, nil
}

// UpdateUser updates user information
func (s *userService) UpdateUser(ctx context.Context, id, email, username, firstName, lastName, status string) (*database.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Update fields
	if email != "" {
		user.Email = email
	}
	if username != "" {
		user.Username = username
	}
	if firstName != "" {
		user.FirstName = firstName
	}
	if lastName != "" {
		user.LastName = lastName
	}
	if status != "" {
		user.Status = status
	}

	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Don't return password hash
	user.Password = ""
	return user, nil
}

// DeleteUser deletes a user
func (s *userService) DeleteUser(ctx context.Context, id string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	return s.userRepo.Delete(ctx, id)
}

// ListUsers lists users with pagination and filtering
func (s *userService) ListUsers(ctx context.Context, page, pageSize int, filter string) ([]*database.User, int64, error) {
	offset := (page - 1) * pageSize
	users, total, err := s.userRepo.List(ctx, offset, pageSize, filter)
	if err != nil {
		return nil, 0, err
	}

	// Don't return password hashes
	for _, user := range users {
		user.Password = ""
	}

	return users, total, nil
}

// AuthenticateUser authenticates user with email and password
func (s *userService) AuthenticateUser(ctx context.Context, email, password string) (*database.User, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", errors.New("invalid credentials")
	}

	// Verify password
	if !s.verifyPassword(password, user.Password) {
		return nil, "", errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := s.generateJWT(user.ID, user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %v", err)
	}

	// Don't return password hash
	user.Password = ""
	return user, token, nil
}

// hashPassword hashes a password using Argon2
func (s *userService) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, 64*1024, 1, 4, b64Salt, b64Hash), nil
}

// verifyPassword verifies a password against its hash
func (s *userService) verifyPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	otherHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	return subtle.ConstantTimeCompare(hash, otherHash) == 1
}

// generateJWT generates a JWT token for the user
func (s *userService) generateJWT(userID, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}