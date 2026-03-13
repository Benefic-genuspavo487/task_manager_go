package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/loks1k192/task-manager/internal/domain"
	"github.com/loks1k192/task-manager/internal/usecase"
)

func setupTeamTest() (*usecase.TeamUseCase, *mockTeamRepo, *mockUserRepo, *mockEmailService) {
	teamRepo := newMockTeamRepo()
	userRepo := newMockUserRepo()
	emailSvc := &mockEmailService{}

	userRepo.Create(context.Background(), &domain.User{
		Email:    "owner@test.com",
		Username: "owner",
	})
	userRepo.Create(context.Background(), &domain.User{
		Email:    "invitee@test.com",
		Username: "invitee",
	})

	uc := usecase.NewTeamUseCase(teamRepo, userRepo, emailSvc)
	return uc, teamRepo, userRepo, emailSvc
}

func TestCreateTeam_Success(t *testing.T) {
	uc, _, _, _ := setupTeamTest()
	team, err := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "My Team"}, 1)
	require.NoError(t, err)
	assert.Equal(t, "My Team", team.Name)
	assert.Equal(t, int64(1), team.CreatedBy)
}

func TestCreateTeam_OwnerBecomesMember(t *testing.T) {
	uc, teamRepo, _, _ := setupTeamTest()
	team, err := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team"}, 1)
	require.NoError(t, err)

	member, err := teamRepo.GetMember(context.Background(), 1, team.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleOwner, member.Role)
}

func TestInvite_Success(t *testing.T) {
	uc, _, _, emailSvc := setupTeamTest()
	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team"}, 1)

	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 2}, 1)
	require.NoError(t, err)
	assert.Len(t, emailSvc.sent, 1)
	assert.Equal(t, "invitee@test.com", emailSvc.sent[0])
}

func TestInvite_NonOwnerFails(t *testing.T) {
	uc, teamRepo, _, _ := setupTeamTest()
	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team"}, 1)

	teamRepo.AddMember(context.Background(), &domain.TeamMember{
		UserID: 2, TeamID: team.ID, Role: domain.RoleMember,
	})

	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 1}, 2)
	assert.ErrorIs(t, err, domain.ErrInsufficientRole)
}

func TestInvite_NotTeamMember(t *testing.T) {
	uc, _, _, _ := setupTeamTest()
	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team"}, 1)

	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 1}, 2)
	assert.ErrorIs(t, err, domain.ErrNotTeamMember)
}

func TestInvite_AlreadyMember(t *testing.T) {
	uc, _, _, _ := setupTeamTest()
	team, _ := uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team"}, 1)

	_ = uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 2}, 1)

	err := uc.Invite(context.Background(), team.ID, usecase.InviteInput{UserID: 2}, 1)
	assert.ErrorIs(t, err, domain.ErrConflict)
}

func TestListByUser(t *testing.T) {
	uc, _, _, _ := setupTeamTest()
	uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team 1"}, 1)
	uc.Create(context.Background(), usecase.CreateTeamInput{Name: "Team 2"}, 1)

	teams, err := uc.ListByUser(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, teams, 2)
}
