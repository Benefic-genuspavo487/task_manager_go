package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/loks1k192/task-manager/internal/usecase"
)

func newAuthUC() (*usecase.AuthUseCase, *mockUserRepo) {
	repo := newMockUserRepo()
	uc := usecase.NewAuthUseCase(repo, "test-secret", 24*time.Hour)
	return uc, repo
}

func TestRegister_Success(t *testing.T) {
	uc, _ := newAuthUC()
	resp, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "test@example.com", resp.User.Email)
	assert.Equal(t, "testuser", resp.User.Username)
	assert.Equal(t, int64(1), resp.User.ID)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	uc, _ := newAuthUC()
	input := usecase.RegisterInput{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}
	_, err := uc.Register(context.Background(), input)
	require.NoError(t, err)

	input.Username = "another"
	_, err = uc.Register(context.Background(), input)
	assert.Error(t, err)
}

func TestLogin_Success(t *testing.T) {
	uc, _ := newAuthUC()
	_, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	})
	require.NoError(t, err)

	resp, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "test@example.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
}

func TestLogin_WrongPassword(t *testing.T) {
	uc, _ := newAuthUC()
	_, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	})
	require.NoError(t, err)

	_, err = uc.Login(context.Background(), usecase.LoginInput{
		Email:    "test@example.com",
		Password: "wrongpassword",
	})
	assert.Error(t, err)
}

func TestLogin_UserNotFound(t *testing.T) {
	uc, _ := newAuthUC()
	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})
	assert.Error(t, err)
}

func TestValidateToken_Success(t *testing.T) {
	uc, _ := newAuthUC()
	resp, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	})
	require.NoError(t, err)

	userID, err := uc.ValidateToken(resp.Token)
	require.NoError(t, err)
	assert.Equal(t, int64(1), userID)
}

func TestValidateToken_Invalid(t *testing.T) {
	uc, _ := newAuthUC()
	_, err := uc.ValidateToken("invalid-token")
	assert.Error(t, err)
}
