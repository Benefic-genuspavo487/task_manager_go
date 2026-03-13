package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type TeamRepository interface {
	Create(ctx context.Context, team *Team) error
	GetByID(ctx context.Context, id int64) (*Team, error)
	ListByUser(ctx context.Context, userID int64) ([]Team, error)
	AddMember(ctx context.Context, member *TeamMember) error
	GetMember(ctx context.Context, userID, teamID int64) (*TeamMember, error)
	IsMember(ctx context.Context, userID, teamID int64) (bool, error)
	GetStats(ctx context.Context) ([]TeamStats, error)
	GetTopCreators(ctx context.Context, year int, month int) ([]TopCreator, error)
}

type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	Update(ctx context.Context, task *Task) error
	List(ctx context.Context, filter TaskFilter) ([]Task, error)
	AddHistory(ctx context.Context, h *TaskHistory) error
	GetHistory(ctx context.Context, taskID int64) ([]TaskHistory, error)
	FindOrphaned(ctx context.Context) ([]OrphanedTask, error)
}

type TaskCommentRepository interface {
	Create(ctx context.Context, comment *TaskComment) error
	ListByTask(ctx context.Context, taskID int64) ([]TaskComment, error)
}

type CacheRepository interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) error
}
