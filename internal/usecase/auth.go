package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/loks1k192/task-manager/internal/domain"
)

type AuthUseCase struct {
	userRepo  domain.UserRepository
	jwtSecret []byte
	jwtExp    time.Duration
}

func NewAuthUseCase(userRepo domain.UserRepository, jwtSecret string, jwtExp time.Duration) *AuthUseCase {
	return &AuthUseCase{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		jwtExp:    jwtExp,
	}
}

type RegisterInput struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	existing, _ := uc.userRepo.GetByEmail(ctx, input.Email)
	if existing != nil {
		return nil, domain.ErrConflict
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	user := &domain.User{
		Email:        input.Email,
		Username:     input.Username,
		PasswordHash: string(hash),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token := uc.signToken(user.ID)
	return &AuthResponse{Token: token, User: user}, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	token := uc.signToken(user.ID)
	return &AuthResponse{Token: token, User: user}, nil
}

func (uc *AuthUseCase) signToken(userID int64) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(uc.jwtExp).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString(uc.jwtSecret)
	return signed
}

func (uc *AuthUseCase) ValidateToken(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrUnauthorized
		}
		return uc.jwtSecret, nil
	})
	if err != nil {
		return 0, domain.ErrUnauthorized
	}

	claims := token.Claims.(jwt.MapClaims)

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, domain.ErrUnauthorized
	}

	return int64(userIDFloat), nil
}
