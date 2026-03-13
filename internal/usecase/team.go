package usecase

import (
	"context"
	"errors"

	"github.com/loks1k192/task-manager/internal/domain"
)

type TeamUseCase struct {
	teamRepo domain.TeamRepository
	userRepo domain.UserRepository
	email    EmailService
}

type EmailService interface {
	SendInvite(ctx context.Context, email, teamName string) error
}

func NewTeamUseCase(teamRepo domain.TeamRepository, userRepo domain.UserRepository, email EmailService) *TeamUseCase {
	return &TeamUseCase{
		teamRepo: teamRepo,
		userRepo: userRepo,
		email:    email,
	}
}

type CreateTeamInput struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type InviteInput struct {
	UserID int64 `json:"user_id" validate:"required"`
}

func (uc *TeamUseCase) Create(ctx context.Context, input CreateTeamInput, creatorID int64) (*domain.Team, error) {
	team := &domain.Team{
		Name:      input.Name,
		CreatedBy: creatorID,
	}

	if err := uc.teamRepo.Create(ctx, team); err != nil {
		return nil, err
	}

	member := &domain.TeamMember{
		UserID: creatorID,
		TeamID: team.ID,
		Role:   domain.RoleOwner,
	}
	if err := uc.teamRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return team, nil
}

func (uc *TeamUseCase) ListByUser(ctx context.Context, userID int64) ([]domain.Team, error) {
	return uc.teamRepo.ListByUser(ctx, userID)
}

func (uc *TeamUseCase) Invite(ctx context.Context, teamID int64, input InviteInput, inviterID int64) error {
	inviter, err := uc.teamRepo.GetMember(ctx, inviterID, teamID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotTeamMember
		}
		return err
	}
	if inviter.Role != domain.RoleOwner && inviter.Role != domain.RoleAdmin {
		return domain.ErrInsufficientRole
	}

	invitee, err := uc.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return err
	}

	isMember, err := uc.teamRepo.IsMember(ctx, input.UserID, teamID)
	if err != nil {
		return err
	}
	if isMember {
		return domain.ErrConflict
	}

	member := &domain.TeamMember{
		UserID: input.UserID,
		TeamID: teamID,
		Role:   domain.RoleMember,
	}
	if err := uc.teamRepo.AddMember(ctx, member); err != nil {
		return err
	}

	team, err := uc.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil // Member already added, email failure is non-critical.
	}
	_ = uc.email.SendInvite(ctx, invitee.Email, team.Name)

	return nil
}

func (uc *TeamUseCase) GetStats(ctx context.Context) ([]domain.TeamStats, error) {
	return uc.teamRepo.GetStats(ctx)
}

func (uc *TeamUseCase) GetTopCreators(ctx context.Context, year, month int) ([]domain.TopCreator, error) {
	return uc.teamRepo.GetTopCreators(ctx, year, month)
}
