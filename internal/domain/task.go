package domain

import "time"

type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

type Task struct {
	ID          int64        `json:"id" db:"id"`
	Title       string       `json:"title" db:"title"`
	Description string       `json:"description" db:"description"`
	Status      TaskStatus   `json:"status" db:"status"`
	Priority    TaskPriority `json:"priority" db:"priority"`
	AssigneeID  *int64       `json:"assignee_id" db:"assignee_id"`
	TeamID      int64        `json:"team_id" db:"team_id"`
	CreatedBy   int64        `json:"created_by" db:"created_by"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

type TaskHistory struct {
	ID        int64     `json:"id" db:"id"`
	TaskID    int64     `json:"task_id" db:"task_id"`
	ChangedBy int64     `json:"changed_by" db:"changed_by"`
	Field     string    `json:"field" db:"field_name"`
	OldValue  string    `json:"old_value" db:"old_value"`
	NewValue  string    `json:"new_value" db:"new_value"`
	ChangedAt time.Time `json:"changed_at" db:"changed_at"`
}

type TaskComment struct {
	ID        int64     `json:"id" db:"id"`
	TaskID    int64     `json:"task_id" db:"task_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Body      string    `json:"body" db:"body"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type TaskFilter struct {
	TeamID     *int64      `json:"team_id"`
	Status     *TaskStatus `json:"status"`
	AssigneeID *int64      `json:"assignee_id"`
	Cursor     *int64      `json:"cursor"`
	Limit      int         `json:"limit"`
}

type OrphanedTask struct {
	TaskID       int64  `json:"task_id" db:"task_id"`
	TaskTitle    string `json:"task_title" db:"task_title"`
	AssigneeID   int64  `json:"assignee_id" db:"assignee_id"`
	AssigneeName string `json:"assignee_name" db:"assignee_name"`
	TeamID       int64  `json:"team_id" db:"team_id"`
	TeamName     string `json:"team_name" db:"team_name"`
}
