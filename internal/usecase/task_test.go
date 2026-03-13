package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/loks1k192/task-manager/internal/domain"
	"github.com/loks1k192/task-manager/internal/usecase"
)

func setupTaskTest() (*usecase.TaskUseCase, *mockTaskRepo, *mockTeamRepo) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	commentRepo := newMockCommentRepo()
	cacheRepo := newMockCacheRepo()

	teamRepo.Create(context.Background(), &domain.Team{Name: "Team1", CreatedBy: 1})
	teamRepo.AddMember(context.Background(), &domain.TeamMember{
		UserID: 1, TeamID: 1, Role: domain.RoleOwner,
	})

	uc := usecase.NewTaskUseCase(taskRepo, teamRepo, commentRepo, cacheRepo)
	return uc, taskRepo, teamRepo
}

func TestCreateTask_Success(t *testing.T) {
	uc, _, _ := setupTaskTest()
	task, err := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title:    "Test Task",
		Priority: "high",
		TeamID:   1,
	}, 1)
	require.NoError(t, err)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, domain.StatusTodo, task.Status)
	assert.Equal(t, domain.TaskPriority("high"), task.Priority)
}

func TestCreateTask_NotTeamMember(t *testing.T) {
	uc, _, _ := setupTaskTest()
	_, err := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title:    "Test Task",
		Priority: "medium",
		TeamID:   1,
	}, 999) // non-member
	assert.ErrorIs(t, err, domain.ErrNotTeamMember)
}

func TestUpdateTask_Success(t *testing.T) {
	uc, _, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title:    "Original",
		Priority: "low",
		TeamID:   1,
	}, 1)

	newTitle := "Updated Title"
	newStatus := "in_progress"
	updated, err := uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{
		Title:  &newTitle,
		Status: &newStatus,
	}, 1)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)
	assert.Equal(t, domain.StatusInProgress, updated.Status)
}

func TestUpdateTask_RecordsHistory(t *testing.T) {
	uc, taskRepo, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title:    "Original",
		Priority: "low",
		TeamID:   1,
	}, 1)

	newTitle := "Changed"
	uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{
		Title: &newTitle,
	}, 1)

	history := taskRepo.history[task.ID]
	require.Len(t, history, 1)
	assert.Equal(t, "title", history[0].Field)
	assert.Equal(t, "Original", history[0].OldValue)
	assert.Equal(t, "Changed", history[0].NewValue)
}

func TestUpdateTask_NotTeamMember(t *testing.T) {
	uc, _, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title:    "Task",
		Priority: "medium",
		TeamID:   1,
	}, 1)

	newTitle := "Hacked"
	_, err := uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{
		Title: &newTitle,
	}, 999)
	assert.ErrorIs(t, err, domain.ErrNotTeamMember)
}

func TestUpdateTask_NotFound(t *testing.T) {
	uc, _, _ := setupTaskTest()
	newTitle := "New"
	_, err := uc.Update(context.Background(), 999, usecase.UpdateTaskInput{
		Title: &newTitle,
	}, 1)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetHistory(t *testing.T) {
	uc, _, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title:    "Task",
		Priority: "medium",
		TeamID:   1,
	}, 1)

	newTitle := "Updated"
	newStatus := "done"
	uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{
		Title:  &newTitle,
		Status: &newStatus,
	}, 1)

	history, err := uc.GetHistory(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Len(t, history, 2) // title + status changes
}

func TestCreateComment_Success(t *testing.T) {
	uc, _, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task", Priority: "medium", TeamID: 1,
	}, 1)

	comment, err := uc.CreateComment(context.Background(), task.ID, usecase.CreateCommentInput{
		Body: "Great work!",
	}, 1)
	require.NoError(t, err)
	assert.Equal(t, "Great work!", comment.Body)
}

func TestCreateComment_NotTeamMember(t *testing.T) {
	uc, _, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task", Priority: "medium", TeamID: 1,
	}, 1)

	_, err := uc.CreateComment(context.Background(), task.ID, usecase.CreateCommentInput{
		Body: "Hacker comment",
	}, 999)
	assert.ErrorIs(t, err, domain.ErrNotTeamMember)
}

func TestListComments(t *testing.T) {
	uc, _, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task", Priority: "medium", TeamID: 1,
	}, 1)

	uc.CreateComment(context.Background(), task.ID, usecase.CreateCommentInput{Body: "Comment 1"}, 1)
	uc.CreateComment(context.Background(), task.ID, usecase.CreateCommentInput{Body: "Comment 2"}, 1)

	comments, err := uc.ListComments(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Len(t, comments, 2)
}

func TestListTasks(t *testing.T) {
	uc, _, _ := setupTaskTest()
	uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task 1", Priority: "high", TeamID: 1,
	}, 1)
	uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task 2", Priority: "low", TeamID: 1,
	}, 1)

	tasks, err := uc.List(context.Background(), domain.TaskFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}
