package mysql

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/loks1k192/task-manager/internal/domain"
)

type commentRepo struct {
	db *sqlx.DB
}

func NewCommentRepository(db *sqlx.DB) domain.TaskCommentRepository {
	return &commentRepo{db: db}
}

func (r *commentRepo) Create(ctx context.Context, c *domain.TaskComment) error {
	const q = `INSERT INTO task_comments (task_id, user_id, body) VALUES (?, ?, ?)`
	result, err := r.db.ExecContext(ctx, q, c.TaskID, c.UserID, c.Body)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	return nil
}

func (r *commentRepo) ListByTask(ctx context.Context, taskID int64) ([]domain.TaskComment, error) {
	var comments []domain.TaskComment
	const q = `SELECT id, task_id, user_id, body, created_at FROM task_comments WHERE task_id = ? ORDER BY created_at ASC`
	if err := r.db.SelectContext(ctx, &comments, q, taskID); err != nil {
		return nil, err
	}
	return comments, nil
}
