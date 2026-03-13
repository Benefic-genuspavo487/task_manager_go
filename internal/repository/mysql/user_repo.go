package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/loks1k192/task-manager/internal/domain"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) domain.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	const q = `INSERT INTO users (email, username, password_hash) VALUES (?, ?, ?)`
	result, err := r.db.ExecContext(ctx, q, user.Email, user.Username, user.PasswordHash)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var user domain.User
	const q = `SELECT id, email, username, password_hash, created_at, updated_at FROM users WHERE id = ?`
	if err := r.db.GetContext(ctx, &user, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	const q = `SELECT id, email, username, password_hash, created_at, updated_at FROM users WHERE email = ?`
	if err := r.db.GetContext(ctx, &user, q, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
