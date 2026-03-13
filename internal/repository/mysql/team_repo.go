package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/loks1k192/task-manager/internal/domain"
)

type teamRepo struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) domain.TeamRepository {
	return &teamRepo{db: db}
}

func (r *teamRepo) Create(ctx context.Context, team *domain.Team) error {
	const q = `INSERT INTO teams (name, created_by) VALUES (?, ?)`
	result, err := r.db.ExecContext(ctx, q, team.Name, team.CreatedBy)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	team.ID = id
	return nil
}

func (r *teamRepo) GetByID(ctx context.Context, id int64) (*domain.Team, error) {
	var team domain.Team
	const q = `SELECT id, name, created_by, created_at, updated_at FROM teams WHERE id = ?`
	if err := r.db.GetContext(ctx, &team, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &team, nil
}

func (r *teamRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Team, error) {
	var teams []domain.Team
	const q = `
		SELECT t.id, t.name, t.created_by, t.created_at, t.updated_at
		FROM teams t
		INNER JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC`
	if err := r.db.SelectContext(ctx, &teams, q, userID); err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *teamRepo) AddMember(ctx context.Context, member *domain.TeamMember) error {
	const q = `INSERT INTO team_members (user_id, team_id, role) VALUES (?, ?, ?)`
	result, err := r.db.ExecContext(ctx, q, member.UserID, member.TeamID, member.Role)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	member.ID = id
	return nil
}

func (r *teamRepo) GetMember(ctx context.Context, userID, teamID int64) (*domain.TeamMember, error) {
	var m domain.TeamMember
	const q = `SELECT id, user_id, team_id, role FROM team_members WHERE user_id = ? AND team_id = ?`
	if err := r.db.GetContext(ctx, &m, q, userID, teamID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *teamRepo) IsMember(ctx context.Context, userID, teamID int64) (bool, error) {
	var count int
	const q = `SELECT COUNT(*) FROM team_members WHERE user_id = ? AND team_id = ?`
	if err := r.db.GetContext(ctx, &count, q, userID, teamID); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teamRepo) GetStats(ctx context.Context) ([]domain.TeamStats, error) {
	var stats []domain.TeamStats
	const q = `
		SELECT
			t.id                                      AS team_id,
			t.name                                    AS team_name,
			COUNT(DISTINCT tm.user_id)                AS member_count,
			COUNT(DISTINCT CASE
				WHEN tk.status = 'done'
				 AND tk.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
				THEN tk.id
			END)                                      AS done_last_7
		FROM teams t
		LEFT JOIN team_members tm ON t.id = tm.team_id
		LEFT JOIN tasks tk        ON t.id = tk.team_id
		GROUP BY t.id, t.name
		ORDER BY t.id`
	if err := r.db.SelectContext(ctx, &stats, q); err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *teamRepo) GetTopCreators(ctx context.Context, year int, month int) ([]domain.TopCreator, error) {
	var top []domain.TopCreator
	const q = `
		SELECT team_id, team_name, user_id, username, tasks_created, rnk
		FROM (
			SELECT
				t.id                                           AS team_id,
				t.name                                         AS team_name,
				u.id                                           AS user_id,
				u.username                                     AS username,
				COUNT(tk.id)                                   AS tasks_created,
				ROW_NUMBER() OVER (
					PARTITION BY t.id
					ORDER BY COUNT(tk.id) DESC
				)                                              AS rnk
			FROM tasks tk
			INNER JOIN teams t ON tk.team_id = t.id
			INNER JOIN users u ON tk.created_by = u.id
			WHERE YEAR(tk.created_at) = ? AND MONTH(tk.created_at) = ?
			GROUP BY t.id, t.name, u.id, u.username
		) ranked
		WHERE rnk <= 3
		ORDER BY team_id, rnk`
	if err := r.db.SelectContext(ctx, &top, q, year, month); err != nil {
		return nil, err
	}
	return top, nil
}
