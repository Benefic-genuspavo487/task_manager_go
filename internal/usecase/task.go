package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/loks1k192/task-manager/internal/domain"
)

const tasksCachePrefix = "tasks:team:"
const tasksCacheTTL = 300 // 5 minutes

type TaskUseCase struct {
	taskRepo    domain.TaskRepository
	teamRepo    domain.TeamRepository
	commentRepo domain.TaskCommentRepository
	cache       domain.CacheRepository
}

func NewTaskUseCase(
	taskRepo domain.TaskRepository,
	teamRepo domain.TeamRepository,
	commentRepo domain.TaskCommentRepository,
	cache domain.CacheRepository,
) *TaskUseCase {
	return &TaskUseCase{
		taskRepo:    taskRepo,
		teamRepo:    teamRepo,
		commentRepo: commentRepo,
		cache:       cache,
	}
}

type CreateTaskInput struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=5000"`
	Priority    string `json:"priority" validate:"required,oneof=low medium high"`
	AssigneeID  *int64 `json:"assignee_id"`
	TeamID      int64  `json:"team_id" validate:"required"`
}

type UpdateTaskInput struct {
	Title       *string `json:"title" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description" validate:"omitempty,max=5000"`
	Status      *string `json:"status" validate:"omitempty,oneof=todo in_progress done"`
	Priority    *string `json:"priority" validate:"omitempty,oneof=low medium high"`
	AssigneeID  *int64  `json:"assignee_id"`
}

type CreateCommentInput struct {
	Body string `json:"body" validate:"required,min=1,max=2000"`
}

func (uc *TaskUseCase) Create(ctx context.Context, input CreateTaskInput, userID int64) (*domain.Task, error) {
	isMember, err := uc.teamRepo.IsMember(ctx, userID, input.TeamID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotTeamMember
	}

	task := &domain.Task{
		Title:       input.Title,
		Description: input.Description,
		Status:      domain.StatusTodo,
		Priority:    domain.TaskPriority(input.Priority),
		AssigneeID:  input.AssigneeID,
		TeamID:      input.TeamID,
		CreatedBy:   userID,
	}

	if err := uc.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	_ = uc.cache.DeleteByPrefix(ctx, tasksCachePrefix+strconv.FormatInt(input.TeamID, 10))

	return task, nil
}

func (uc *TaskUseCase) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	return uc.taskRepo.GetByID(ctx, id)
}

func (uc *TaskUseCase) Update(ctx context.Context, taskID int64, input UpdateTaskInput, userID int64) (*domain.Task, error) {
	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	isMember, err := uc.teamRepo.IsMember(ctx, userID, task.TeamID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotTeamMember
	}

	var changes []domain.TaskHistory

	if input.Title != nil && *input.Title != task.Title {
		changes = append(changes, domain.TaskHistory{
			TaskID: taskID, ChangedBy: userID, Field: "title",
			OldValue: task.Title, NewValue: *input.Title,
		})
		task.Title = *input.Title
	}
	if input.Description != nil && *input.Description != task.Description {
		changes = append(changes, domain.TaskHistory{
			TaskID: taskID, ChangedBy: userID, Field: "description",
			OldValue: task.Description, NewValue: *input.Description,
		})
		task.Description = *input.Description
	}
	if input.Status != nil && domain.TaskStatus(*input.Status) != task.Status {
		changes = append(changes, domain.TaskHistory{
			TaskID: taskID, ChangedBy: userID, Field: "status",
			OldValue: string(task.Status), NewValue: *input.Status,
		})
		task.Status = domain.TaskStatus(*input.Status)
	}
	if input.Priority != nil && domain.TaskPriority(*input.Priority) != task.Priority {
		changes = append(changes, domain.TaskHistory{
			TaskID: taskID, ChangedBy: userID, Field: "priority",
			OldValue: string(task.Priority), NewValue: *input.Priority,
		})
		task.Priority = domain.TaskPriority(*input.Priority)
	}
	if input.AssigneeID != nil {
		oldVal := ""
		if task.AssigneeID != nil {
			oldVal = strconv.FormatInt(*task.AssigneeID, 10)
		}
		newVal := strconv.FormatInt(*input.AssigneeID, 10)
		if oldVal != newVal {
			changes = append(changes, domain.TaskHistory{
				TaskID: taskID, ChangedBy: userID, Field: "assignee_id",
				OldValue: oldVal, NewValue: newVal,
			})
			task.AssigneeID = input.AssigneeID
		}
	}

	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	for i := range changes {
		_ = uc.taskRepo.AddHistory(ctx, &changes[i])
	}

	_ = uc.cache.DeleteByPrefix(ctx, tasksCachePrefix+strconv.FormatInt(task.TeamID, 10))

	return task, nil
}

func (uc *TaskUseCase) List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
	if filter.TeamID != nil && filter.Cursor == nil {
		cacheKey := uc.buildCacheKey(filter)
		cached, err := uc.cache.Get(ctx, cacheKey)
		if err == nil && cached != nil {
			var tasks []domain.Task
			if json.Unmarshal(cached, &tasks) == nil {
				return tasks, nil
			}
		}

		tasks, err := uc.taskRepo.List(ctx, filter)
		if err != nil {
			return nil, err
		}

		if data, err := json.Marshal(tasks); err == nil {
			_ = uc.cache.Set(ctx, cacheKey, data, tasksCacheTTL)
		}
		return tasks, nil
	}

	return uc.taskRepo.List(ctx, filter)
}

func (uc *TaskUseCase) GetHistory(ctx context.Context, taskID int64) ([]domain.TaskHistory, error) {
	return uc.taskRepo.GetHistory(ctx, taskID)
}

func (uc *TaskUseCase) FindOrphaned(ctx context.Context) ([]domain.OrphanedTask, error) {
	return uc.taskRepo.FindOrphaned(ctx)
}

func (uc *TaskUseCase) CreateComment(ctx context.Context, taskID int64, input CreateCommentInput, userID int64) (*domain.TaskComment, error) {
	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	isMember, err := uc.teamRepo.IsMember(ctx, userID, task.TeamID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotTeamMember
	}

	comment := &domain.TaskComment{
		TaskID: taskID,
		UserID: userID,
		Body:   input.Body,
	}
	if err := uc.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (uc *TaskUseCase) ListComments(ctx context.Context, taskID int64) ([]domain.TaskComment, error) {
	return uc.commentRepo.ListByTask(ctx, taskID)
}

func (uc *TaskUseCase) buildCacheKey(filter domain.TaskFilter) string {
	key := tasksCachePrefix
	if filter.TeamID != nil {
		key += strconv.FormatInt(*filter.TeamID, 10)
	}
	if filter.Status != nil {
		key += ":s:" + string(*filter.Status)
	}
	if filter.AssigneeID != nil {
		key += ":a:" + strconv.FormatInt(*filter.AssigneeID, 10)
	}
	key += fmt.Sprintf(":l:%d", filter.Limit)
	return key
}
