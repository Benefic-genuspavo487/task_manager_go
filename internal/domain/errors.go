package domain

import "errors"

var (
	ErrNotFound         = errors.New("resource not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrConflict         = errors.New("resource already exists")
	ErrInvalidInput     = errors.New("invalid input")
	ErrNotTeamMember    = errors.New("user is not a member of this team")
	ErrInsufficientRole = errors.New("insufficient role")
)
