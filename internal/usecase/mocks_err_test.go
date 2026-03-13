package usecase_test

import (
	"context"
	"errors"

	"github.com/loks1k192/task-manager/internal/domain"
)

var errSimulated = errors.New("simulated error")

type errUserRepo struct {
	mockUserRepo
	createErr     error
	getByEmailErr error
}

func (m *errUserRepo) Create(ctx context.Context, user *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	return m.mockUserRepo.Create(ctx, user)
}

func (m *errUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailErr != nil {
		return nil, m.getByEmailErr
	}
	return m.mockUserRepo.GetByEmail(ctx, email)
}

type errTeamRepo struct {
	mockTeamRepo
	createErr    error
	addMemberErr error
	getMemberErr error
	isMemberErr  error
	getByIDErr   error
}

func (m *errTeamRepo) Create(ctx context.Context, team *domain.Team) error {
	if m.createErr != nil {
		return m.createErr
	}
	return m.mockTeamRepo.Create(ctx, team)
}

func (m *errTeamRepo) AddMember(ctx context.Context, member *domain.TeamMember) error {
	if m.addMemberErr != nil {
		return m.addMemberErr
	}
	return m.mockTeamRepo.AddMember(ctx, member)
}

func (m *errTeamRepo) GetMember(ctx context.Context, userID, teamID int64) (*domain.TeamMember, error) {
	if m.getMemberErr != nil {
		return nil, m.getMemberErr
	}
	return m.mockTeamRepo.GetMember(ctx, userID, teamID)
}

func (m *errTeamRepo) IsMember(ctx context.Context, userID, teamID int64) (bool, error) {
	if m.isMemberErr != nil {
		return false, m.isMemberErr
	}
	return m.mockTeamRepo.IsMember(ctx, userID, teamID)
}

func (m *errTeamRepo) GetByID(ctx context.Context, id int64) (*domain.Team, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return m.mockTeamRepo.GetByID(ctx, id)
}

type errTaskRepo struct {
	mockTaskRepo
	createErr error
	updateErr error
	listErr   error
}

func (m *errTaskRepo) Create(ctx context.Context, task *domain.Task) error {
	if m.createErr != nil {
		return m.createErr
	}
	return m.mockTaskRepo.Create(ctx, task)
}

func (m *errTaskRepo) Update(ctx context.Context, task *domain.Task) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.mockTaskRepo.Update(ctx, task)
}

func (m *errTaskRepo) List(_ context.Context, _ domain.TaskFilter) ([]domain.Task, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []domain.Task
	for _, t := range m.tasks {
		result = append(result, *t)
	}
	return result, nil
}

type errCommentRepo struct {
	mockCommentRepo
	createErr error
}

func (m *errCommentRepo) Create(ctx context.Context, c *domain.TaskComment) error {
	if m.createErr != nil {
		return m.createErr
	}
	return m.mockCommentRepo.Create(ctx, c)
}
