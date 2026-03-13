package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/loks1k192/task-manager/internal/domain"
	"github.com/loks1k192/task-manager/internal/usecase"
)

func TestGetByID(t *testing.T) {
	uc, _, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task", Priority: "medium", TeamID: 1,
	}, 1)

	fetched, err := uc.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, fetched.ID)
}

func TestGetByID_NotFound(t *testing.T) {
	uc, _, _ := setupTaskTest()
	_, err := uc.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestFindOrphaned(t *testing.T) {
	uc, _, _ := setupTaskTest()
	result, err := uc.FindOrphaned(context.Background())
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetStats(t *testing.T) {
	uc, _, _, _ := setupTeamTest()
	stats, err := uc.GetStats(context.Background())
	require.NoError(t, err)
	assert.Nil(t, stats)
}

func TestGetTopCreators(t *testing.T) {
	uc, _, _, _ := setupTeamTest()
	top, err := uc.GetTopCreators(context.Background(), 2026, 3)
	require.NoError(t, err)
	assert.Nil(t, top)
}

func TestListTasks_WithTeamFilter_CacheMiss(t *testing.T) {
	uc, _, _ := setupTaskTest()

	uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task 1", Priority: "high", TeamID: 1,
	}, 1)

	teamID := int64(1)
	tasks, err := uc.List(context.Background(), domain.TaskFilter{
		TeamID: &teamID,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tasks)
}

func TestListTasks_WithTeamFilter_CacheHit(t *testing.T) {
	uc, _, _ := setupTaskTest()

	uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task 1", Priority: "high", TeamID: 1,
	}, 1)

	teamID := int64(1)
	filter := domain.TaskFilter{TeamID: &teamID, Limit: 10}

	_, err := uc.List(context.Background(), filter)
	require.NoError(t, err)

	tasks, err := uc.List(context.Background(), filter)
	require.NoError(t, err)
	assert.NotEmpty(t, tasks)
}

func TestListTasks_WithCursor(t *testing.T) {
	uc, _, _ := setupTaskTest()

	uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Task 1", Priority: "high", TeamID: 1,
	}, 1)

	cursor := int64(100)
	teamID := int64(1)
	tasks, err := uc.List(context.Background(), domain.TaskFilter{
		TeamID: &teamID,
		Cursor: &cursor,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.NotNil(t, tasks)
}

func TestUpdateTask_AllFields(t *testing.T) {
	uc, taskRepo, _ := setupTaskTest()

	assignee := int64(1)
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title:       "Original",
		Description: "Desc",
		Priority:    "low",
		TeamID:      1,
		AssigneeID:  &assignee,
	}, 1)

	newTitle := "New Title"
	newDesc := "New Desc"
	newStatus := "done"
	newPriority := "high"
	newAssignee := int64(2)

	updated, err := uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{
		Title:       &newTitle,
		Description: &newDesc,
		Status:      &newStatus,
		Priority:    &newPriority,
		AssigneeID:  &newAssignee,
	}, 1)
	require.NoError(t, err)
	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "New Desc", updated.Description)
	assert.Equal(t, domain.StatusDone, updated.Status)
	assert.Equal(t, domain.TaskPriority("high"), updated.Priority)
	assert.Equal(t, int64(2), *updated.AssigneeID)

	history := taskRepo.history[task.ID]
	assert.Len(t, history, 5) // title, description, status, priority, assignee_id
}

func TestUpdateTask_NoChanges(t *testing.T) {
	uc, taskRepo, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "Same", Priority: "medium", TeamID: 1,
	}, 1)

	sameTitle := "Same"
	_, err := uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{
		Title: &sameTitle,
	}, 1)
	require.NoError(t, err)
	assert.Empty(t, taskRepo.history[task.ID])
}

func TestUpdateTask_AssigneeFromNil(t *testing.T) {
	uc, taskRepo, _ := setupTaskTest()
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "No Assignee", Priority: "medium", TeamID: 1,
	}, 1)

	newAssignee := int64(1)
	_, err := uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{
		AssigneeID: &newAssignee,
	}, 1)
	require.NoError(t, err)
	history := taskRepo.history[task.ID]
	require.Len(t, history, 1)
	assert.Equal(t, "assignee_id", history[0].Field)
	assert.Equal(t, "", history[0].OldValue)
	assert.Equal(t, "1", history[0].NewValue)
}

func TestCreateTeam_ErrorHandling(t *testing.T) {
	uc, _, _, _ := setupTeamTest()

	team, err := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team"}, 1)
	require.NoError(t, err)
	assert.NotNil(t, team)
}

func TestListByUser_Empty(t *testing.T) {
	uc, _, _, _ := setupTeamTest()
	teams, err := uc.ListByUser(context.Background(), 999)
	require.NoError(t, err)
	assert.Empty(t, teams)
}

func TestInvite_UserNotFound(t *testing.T) {
	uc, _, _, _ := setupTeamTest()
	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team"}, 1)

	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 9999}, 1)
	assert.Error(t, err)
}
