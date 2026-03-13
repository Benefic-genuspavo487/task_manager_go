package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/loks1k192/task-manager/internal/domain"
)

type taskRepo struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) domain.TaskRepository {
	return &taskRepo{db: db}
}

func (r *taskRepo) Create(ctx context.Context, task *domain.Task) error {
	const q = `INSERT INTO tasks (title, description, status, priority, assignee_id, team_id, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, q,
		task.Title, task.Description, task.Status, task.Priority,
		task.AssigneeID, task.TeamID, task.CreatedBy,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	task.ID = id
	return nil
}

func (r *taskRepo) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	var task domain.Task
	const q = `SELECT id, title, description, status, priority, assignee_id, team_id, created_by, created_at, updated_at FROM tasks WHERE id = ?`
	if err := r.db.GetContext(ctx, &task, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &task, nil
}

func (r *taskRepo) Update(ctx context.Context, task *domain.Task) error {
	const q = `UPDATE tasks SET title=?, description=?, status=?, priority=?, assignee_id=?, updated_at=NOW() WHERE id=?`
	_, err := r.db.ExecContext(ctx, q,
		task.Title, task.Description, task.Status, task.Priority,
		task.AssigneeID, task.ID,
	)
	return err
}

func (r *taskRepo) List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
	var (
		conditions []string
		args       []interface{}
	)

	if filter.TeamID != nil {
		conditions = append(conditions, "team_id = ?")
		args = append(args, *filter.TeamID)
	}
	if filter.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.AssigneeID != nil {
		conditions = append(conditions, "assignee_id = ?")
		args = append(args, *filter.AssigneeID)
	}
	if filter.Cursor != nil {
		conditions = append(conditions, "id < ?")
		args = append(args, *filter.Cursor)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	q := fmt.Sprintf(`
		SELECT id, title, description, status, priority, assignee_id, team_id, created_by, created_at, updated_at
		FROM tasks
		%s
		ORDER BY id DESC
		LIMIT ?`, where)

	args = append(args, limit)

	var tasks []domain.Task
	if err := r.db.SelectContext(ctx, &tasks, q, args...); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *taskRepo) AddHistory(ctx context.Context, h *domain.TaskHistory) error {
	const q = `INSERT INTO task_history (task_id, changed_by, field_name, old_value, new_value) VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, q, h.TaskID, h.ChangedBy, h.Field, h.OldValue, h.NewValue)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	h.ID = id
	return nil
}

func (r *taskRepo) GetHistory(ctx context.Context, taskID int64) ([]domain.TaskHistory, error) {
	var history []domain.TaskHistory
	const q = `
		SELECT id, task_id, changed_by, field_name, old_value, new_value, changed_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY changed_at DESC`
	if err := r.db.SelectContext(ctx, &history, q, taskID); err != nil {
		return nil, err
	}
	return history, nil
}

func (r *taskRepo) FindOrphaned(ctx context.Context) ([]domain.OrphanedTask, error) {
	var orphaned []domain.OrphanedTask
	const q = `
		SELECT
			tk.id          AS task_id,
			tk.title       AS task_title,
			tk.assignee_id AS assignee_id,
			u.username     AS assignee_name,
			tk.team_id     AS team_id,
			t.name         AS team_name
		FROM tasks tk
		INNER JOIN users u ON tk.assignee_id = u.id
		INNER JOIN teams t ON tk.team_id = t.id
		WHERE tk.assignee_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM team_members tm
			WHERE tm.user_id = tk.assignee_id
			  AND tm.team_id = tk.team_id
		  )
		ORDER BY tk.id`
	if err := r.db.SelectContext(ctx, &orphaned, q); err != nil {
		return nil, err
	}
	return orphaned, nil
}
