package domain

import "time"

type TeamRole string

const (
	RoleOwner  TeamRole = "owner"
	RoleAdmin  TeamRole = "admin"
	RoleMember TeamRole = "member"
)

type Team struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedBy int64     `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type TeamMember struct {
	ID     int64    `json:"id" db:"id"`
	UserID int64    `json:"user_id" db:"user_id"`
	TeamID int64    `json:"team_id" db:"team_id"`
	Role   TeamRole `json:"role" db:"role"`
}

type TeamStats struct {
	TeamID      int64  `json:"team_id" db:"team_id"`
	TeamName    string `json:"team_name" db:"team_name"`
	MemberCount int    `json:"member_count" db:"member_count"`
	DoneLast7   int    `json:"done_last_7_days" db:"done_last_7"`
}

type TopCreator struct {
	TeamID       int64  `json:"team_id" db:"team_id"`
	TeamName     string `json:"team_name" db:"team_name"`
	UserID       int64  `json:"user_id" db:"user_id"`
	Username     string `json:"username" db:"username"`
	TasksCreated int    `json:"tasks_created" db:"tasks_created"`
	Rank         int    `json:"rank" db:"rnk"`
}
