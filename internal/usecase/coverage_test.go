package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/loks1k192/task-manager/internal/domain"
	"github.com/loks1k192/task-manager/internal/usecase"
)

func TestRegister_CreateUserError(t *testing.T) {
	repo := &errUserRepo{mockUserRepo: *newMockUserRepo(), createErr: errSimulated}
	uc := usecase.NewAuthUseCase(repo, "secret", 24*time.Hour)

	_, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email: "a@b.com", Username: "u", Password: "pass123",
	})
	assert.ErrorIs(t, err, errSimulated)
}

func TestLogin_GetByEmailNonNotFoundError(t *testing.T) {
	repo := &errUserRepo{mockUserRepo: *newMockUserRepo(), getByEmailErr: errSimulated}
	uc := usecase.NewAuthUseCase(repo, "secret", 24*time.Hour)

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email: "a@b.com", Password: "pass",
	})
	assert.ErrorIs(t, err, errSimulated)
}

func TestValidateToken_NonHMACSigningMethod(t *testing.T) {
	uc := usecase.NewAuthUseCase(newMockUserRepo(), "secret", 24*time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": float64(1),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := uc.ValidateToken(tokenStr)
	assert.Error(t, err)
}

func TestValidateToken_MissingUserIDClaim(t *testing.T) {
	secret := "test-secret"
	uc := usecase.NewAuthUseCase(newMockUserRepo(), secret, 24*time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(secret))

	_, err := uc.ValidateToken(tokenStr)
	assert.Error(t, err)
}

func TestValidateToken_UserIDWrongType(t *testing.T) {
	secret := "test-secret"
	uc := usecase.NewAuthUseCase(newMockUserRepo(), secret, 24*time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "not-a-number",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(secret))

	_, err := uc.ValidateToken(tokenStr)
	assert.Error(t, err)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	uc := usecase.NewAuthUseCase(newMockUserRepo(), secret, 24*time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1),
		"exp":     time.Now().Add(-time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(secret))

	_, err := uc.ValidateToken(tokenStr)
	assert.Error(t, err)
}

func TestCreateTeam_RepoCreateError(t *testing.T) {
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo(), createErr: errSimulated}
	userRepo := newMockUserRepo()
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, &mockEmailService{})

	_, err := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "T"}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestCreateTeam_AddMemberError(t *testing.T) {
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo(), addMemberErr: errSimulated}
	userRepo := newMockUserRepo()
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, &mockEmailService{})

	_, err := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "T"}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestInvite_GetMemberNonNotFoundError(t *testing.T) {
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo(), getMemberErr: errSimulated}
	userRepo := newMockUserRepo()
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, &mockEmailService{})

	err := uc.Invite(context.Background(), 1, usecase.InviteInput{UserID: 2}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestInvite_IsMemberError(t *testing.T) {
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo()}
	userRepo := newMockUserRepo()
	userRepo.Create(context.Background(), &domain.User{Email: "a@b.com", Username: "a"})
	userRepo.Create(context.Background(), &domain.User{Email: "b@b.com", Username: "b"})
	emailSvc := &mockEmailService{}
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, emailSvc)

	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "T"}, 1)

	teamRepo.isMemberErr = errSimulated
	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 2}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestInvite_AddMemberError(t *testing.T) {
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo()}
	userRepo := newMockUserRepo()
	userRepo.Create(context.Background(), &domain.User{Email: "a@b.com", Username: "a"})
	userRepo.Create(context.Background(), &domain.User{Email: "b@b.com", Username: "b"})
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, &mockEmailService{})

	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "T"}, 1)

	teamRepo.addMemberErr = errSimulated
	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 2}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestInvite_AdminCanInvite(t *testing.T) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	userRepo.Create(context.Background(), &domain.User{Email: "a@b.com", Username: "owner"})
	userRepo.Create(context.Background(), &domain.User{Email: "b@b.com", Username: "admin"})
	userRepo.Create(context.Background(), &domain.User{Email: "c@b.com", Username: "invitee"})
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, &mockEmailService{})

	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "T"}, 1)

	teamRepo.AddMember(context.Background(), &domain.TeamMember{
		UserID: 2, TeamID: team.ID, Role: domain.RoleAdmin,
	})

	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 3}, 2)
	assert.NoError(t, err)
}

func TestCreateTask_IsMemberError(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo(), isMemberErr: errSimulated}
	uc := usecase.NewTaskUseCase(taskRepo, teamRepo, newMockCommentRepo(), newMockCacheRepo())

	_, err := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1,
	}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestCreateTask_RepoCreateError(t *testing.T) {
	taskRepo := &errTaskRepo{mockTaskRepo: *newMockTaskRepo(), createErr: errSimulated}
	teamRepo := newMockTeamRepo()
	teamRepo.Create(context.Background(), &domain.Team{Name: "T", CreatedBy: 1})
	teamRepo.AddMember(context.Background(), &domain.TeamMember{UserID: 1, TeamID: 1, Role: domain.RoleOwner})
	uc := usecase.NewTaskUseCase(taskRepo, teamRepo, newMockCommentRepo(), newMockCacheRepo())

	_, err := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1,
	}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestUpdateTask_IsMemberError(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo()}
	teamRepo.mockTeamRepo.Create(context.Background(), &domain.Team{Name: "T", CreatedBy: 1})
	teamRepo.mockTeamRepo.AddMember(context.Background(), &domain.TeamMember{UserID: 1, TeamID: 1, Role: domain.RoleOwner})
	uc := usecase.NewTaskUseCase(taskRepo, teamRepo, newMockCommentRepo(), newMockCacheRepo())

	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1,
	}, 1)

	teamRepo.isMemberErr = errSimulated
	title := "New"
	_, err := uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{Title: &title}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestUpdateTask_RepoUpdateError(t *testing.T) {
	taskRepo := &errTaskRepo{mockTaskRepo: *newMockTaskRepo()}
	teamRepo := newMockTeamRepo()
	teamRepo.Create(context.Background(), &domain.Team{Name: "T", CreatedBy: 1})
	teamRepo.AddMember(context.Background(), &domain.TeamMember{UserID: 1, TeamID: 1, Role: domain.RoleOwner})
	uc := usecase.NewTaskUseCase(taskRepo, teamRepo, newMockCommentRepo(), newMockCacheRepo())

	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1,
	}, 1)

	taskRepo.updateErr = errSimulated
	title := "New"
	_, err := uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{Title: &title}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestUpdateTask_SameAssigneeNoChange(t *testing.T) {
	uc, taskRepo, _ := setupTaskTest()
	assignee := int64(1)
	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1, AssigneeID: &assignee,
	}, 1)

	sameAssignee := int64(1)
	_, err := uc.Update(context.Background(), task.ID, usecase.UpdateTaskInput{AssigneeID: &sameAssignee}, 1)
	require.NoError(t, err)
	assert.Empty(t, taskRepo.history[task.ID])
}

func TestListTasks_RepoError_CacheBranch(t *testing.T) {
	taskRepo := &errTaskRepo{mockTaskRepo: *newMockTaskRepo(), listErr: errSimulated}
	teamRepo := newMockTeamRepo()
	uc := usecase.NewTaskUseCase(taskRepo, teamRepo, newMockCommentRepo(), newMockCacheRepo())

	teamID := int64(1)
	_, err := uc.List(context.Background(), domain.TaskFilter{TeamID: &teamID, Limit: 10})
	assert.ErrorIs(t, err, errSimulated)
}

func TestListTasks_NilTeamID(t *testing.T) {
	uc, _, _ := setupTaskTest()
	uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1,
	}, 1)

	tasks, err := uc.List(context.Background(), domain.TaskFilter{Limit: 10})
	require.NoError(t, err)
	assert.NotEmpty(t, tasks)
}

func TestListTasks_WithStatusAndAssigneeFilter(t *testing.T) {
	uc, _, _ := setupTaskTest()
	assignee := int64(1)
	uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1, AssigneeID: &assignee,
	}, 1)

	teamID := int64(1)
	status := domain.StatusTodo
	tasks, err := uc.List(context.Background(), domain.TaskFilter{
		TeamID: &teamID, Status: &status, AssigneeID: &assignee, Limit: 10,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tasks)
}

func TestCreateComment_TaskNotFound(t *testing.T) {
	uc, _, _ := setupTaskTest()
	_, err := uc.CreateComment(context.Background(), 999, usecase.CreateCommentInput{Body: "Hi"}, 1)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCreateComment_IsMemberError(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo()}
	teamRepo.mockTeamRepo.Create(context.Background(), &domain.Team{Name: "T", CreatedBy: 1})
	teamRepo.mockTeamRepo.AddMember(context.Background(), &domain.TeamMember{UserID: 1, TeamID: 1, Role: domain.RoleOwner})
	uc := usecase.NewTaskUseCase(taskRepo, teamRepo, newMockCommentRepo(), newMockCacheRepo())

	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1,
	}, 1)

	teamRepo.isMemberErr = errSimulated
	_, err := uc.CreateComment(context.Background(), task.ID, usecase.CreateCommentInput{Body: "Hi"}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestCreateComment_RepoCreateError(t *testing.T) {
	taskRepo := newMockTaskRepo()
	teamRepo := newMockTeamRepo()
	teamRepo.Create(context.Background(), &domain.Team{Name: "T", CreatedBy: 1})
	teamRepo.AddMember(context.Background(), &domain.TeamMember{UserID: 1, TeamID: 1, Role: domain.RoleOwner})
	commentRepo := &errCommentRepo{mockCommentRepo: *newMockCommentRepo(), createErr: errSimulated}
	uc := usecase.NewTaskUseCase(taskRepo, teamRepo, commentRepo, newMockCacheRepo())

	task, _ := uc.Create(context.Background(), usecase.CreateTaskInput{
		Title: "T", Priority: "high", TeamID: 1,
	}, 1)

	_, err := uc.CreateComment(context.Background(), task.ID, usecase.CreateCommentInput{Body: "Hi"}, 1)
	assert.ErrorIs(t, err, errSimulated)
}

func TestInvite_GetByIDErrorAfterAddMember(t *testing.T) {
	teamRepo := &errTeamRepo{mockTeamRepo: *newMockTeamRepo()}
	userRepo := newMockUserRepo()
	userRepo.Create(context.Background(), &domain.User{Email: "a@b.com", Username: "owner"})
	userRepo.Create(context.Background(), &domain.User{Email: "b@b.com", Username: "invitee"})
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, &mockEmailService{})

	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "T"}, 1)

	teamRepo.getByIDErr = errSimulated
	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 2}, 1)
	assert.NoError(t, err)
}
