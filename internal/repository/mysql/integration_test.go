package mysql_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"

	"github.com/loks1k192/task-manager/internal/domain"
	mysqlrepo "github.com/loks1k192/task-manager/internal/repository/mysql"
)

var testDB *sqlx.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("root"),
		mysql.WithPassword("testpass"),
	)
	if err != nil {
		fmt.Printf("failed to start mysql container: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = testcontainers.TerminateContainer(container)
	}()

	connStr, err := container.ConnectionString(ctx, "parseTime=true", "multiStatements=true")
	if err != nil {
		fmt.Printf("failed to get connection string: %v\n", err)
		os.Exit(1)
	}

	testDB, err = sqlx.Connect("mysql", connStr)
	if err != nil {
		fmt.Printf("failed to connect to test db: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	migration, err := os.ReadFile("../../../migrations/001_schema.up.sql")
	if err != nil {
		fmt.Printf("failed to read migration: %v\n", err)
		os.Exit(1)
	}
	if _, err := testDB.Exec(string(migration)); err != nil {
		fmt.Printf("failed to run migration: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func cleanTables(t *testing.T) {
	t.Helper()
	tables := []string{"task_comments", "task_history", "tasks", "team_members", "teams", "users"}
	for _, table := range tables {
		_, err := testDB.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
}

func TestUserRepo_Integration(t *testing.T) {
	cleanTables(t)
	repo := mysqlrepo.NewUserRepository(testDB)
	ctx := context.Background()

	user := &domain.User{
		Email:        "integration@test.com",
		Username:     "integrationuser",
		PasswordHash: "hashed",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotZero(t, user.ID)

	fetched, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "integration@test.com", fetched.Email)

	fetched, err = repo.GetByEmail(ctx, "integration@test.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, fetched.ID)

	_, err = repo.GetByEmail(ctx, "nonexistent@test.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestTeamRepo_Integration(t *testing.T) {
	cleanTables(t)
	userRepo := mysqlrepo.NewUserRepository(testDB)
	teamRepo := mysqlrepo.NewTeamRepository(testDB)
	ctx := context.Background()

	user := &domain.User{Email: "owner@test.com", Username: "owner", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(ctx, user))

	team := &domain.Team{Name: "Integration Team", CreatedBy: user.ID}
	require.NoError(t, teamRepo.Create(ctx, team))
	assert.NotZero(t, team.ID)

	member := &domain.TeamMember{UserID: user.ID, TeamID: team.ID, Role: domain.RoleOwner}
	require.NoError(t, teamRepo.AddMember(ctx, member))

	isMember, err := teamRepo.IsMember(ctx, user.ID, team.ID)
	require.NoError(t, err)
	assert.True(t, isMember)

	teams, err := teamRepo.ListByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 1)
}

func TestTaskRepo_CursorPagination(t *testing.T) {
	cleanTables(t)
	userRepo := mysqlrepo.NewUserRepository(testDB)
	teamRepo := mysqlrepo.NewTeamRepository(testDB)
	taskRepo := mysqlrepo.NewTaskRepository(testDB)
	ctx := context.Background()

	user := &domain.User{Email: "u@t.com", Username: "u", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(ctx, user))
	team := &domain.Team{Name: "T", CreatedBy: user.ID}
	require.NoError(t, teamRepo.Create(ctx, team))

	for i := 0; i < 5; i++ {
		task := &domain.Task{
			Title: fmt.Sprintf("Task %d", i), Status: domain.StatusTodo,
			Priority: domain.PriorityMedium, TeamID: team.ID, CreatedBy: user.ID,
		}
		require.NoError(t, taskRepo.Create(ctx, task))
	}

	tasks, err := taskRepo.List(ctx, domain.TaskFilter{TeamID: &team.ID, Limit: 3})
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	cursor := tasks[2].ID
	tasks2, err := taskRepo.List(ctx, domain.TaskFilter{TeamID: &team.ID, Limit: 3, Cursor: &cursor})
	require.NoError(t, err)
	assert.Len(t, tasks2, 2)
}

func TestTaskRepo_History(t *testing.T) {
	cleanTables(t)
	userRepo := mysqlrepo.NewUserRepository(testDB)
	teamRepo := mysqlrepo.NewTeamRepository(testDB)
	taskRepo := mysqlrepo.NewTaskRepository(testDB)
	ctx := context.Background()

	user := &domain.User{Email: "u@t.com", Username: "u", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(ctx, user))
	team := &domain.Team{Name: "T", CreatedBy: user.ID}
	require.NoError(t, teamRepo.Create(ctx, team))

	task := &domain.Task{
		Title: "Original", Status: domain.StatusTodo,
		Priority: domain.PriorityLow, TeamID: team.ID, CreatedBy: user.ID,
	}
	require.NoError(t, taskRepo.Create(ctx, task))

	h := &domain.TaskHistory{
		TaskID: task.ID, ChangedBy: user.ID,
		Field: "status", OldValue: "todo", NewValue: "done",
	}
	require.NoError(t, taskRepo.AddHistory(ctx, h))

	history, err := taskRepo.GetHistory(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "status", history[0].Field)
}

func TestTeamRepo_GetStats(t *testing.T) {
	cleanTables(t)
	userRepo := mysqlrepo.NewUserRepository(testDB)
	teamRepo := mysqlrepo.NewTeamRepository(testDB)
	taskRepo := mysqlrepo.NewTaskRepository(testDB)
	ctx := context.Background()

	user := &domain.User{Email: "u@t.com", Username: "u", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(ctx, user))

	team := &domain.Team{Name: "Stats Team", CreatedBy: user.ID}
	require.NoError(t, teamRepo.Create(ctx, team))
	require.NoError(t, teamRepo.AddMember(ctx, &domain.TeamMember{
		UserID: user.ID, TeamID: team.ID, Role: domain.RoleOwner,
	}))

	task := &domain.Task{
		Title: "Done Task", Status: domain.StatusDone,
		Priority: domain.PriorityHigh, TeamID: team.ID, CreatedBy: user.ID,
	}
	require.NoError(t, taskRepo.Create(ctx, task))

	stats, err := teamRepo.GetStats(ctx)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, "Stats Team", stats[0].TeamName)
	assert.Equal(t, 1, stats[0].MemberCount)
	assert.Equal(t, 1, stats[0].DoneLast7)
}

func TestTeamRepo_TopCreators(t *testing.T) {
	cleanTables(t)
	userRepo := mysqlrepo.NewUserRepository(testDB)
	teamRepo := mysqlrepo.NewTeamRepository(testDB)
	taskRepo := mysqlrepo.NewTaskRepository(testDB)
	ctx := context.Background()

	user := &domain.User{Email: "u@t.com", Username: "topcreator", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(ctx, user))

	team := &domain.Team{Name: "Top Team", CreatedBy: user.ID}
	require.NoError(t, teamRepo.Create(ctx, team))

	for i := 0; i < 3; i++ {
		require.NoError(t, taskRepo.Create(ctx, &domain.Task{
			Title: fmt.Sprintf("T%d", i), Status: domain.StatusTodo,
			Priority: domain.PriorityMedium, TeamID: team.ID, CreatedBy: user.ID,
		}))
	}

	now := time.Now()
	top, err := teamRepo.GetTopCreators(ctx, now.Year(), int(now.Month()))
	require.NoError(t, err)
	require.Len(t, top, 1)
	assert.Equal(t, "topcreator", top[0].Username)
	assert.Equal(t, 3, top[0].TasksCreated)
}

func TestTaskRepo_FindOrphaned(t *testing.T) {
	cleanTables(t)
	userRepo := mysqlrepo.NewUserRepository(testDB)
	teamRepo := mysqlrepo.NewTeamRepository(testDB)
	taskRepo := mysqlrepo.NewTaskRepository(testDB)
	ctx := context.Background()

	owner := &domain.User{Email: "owner@t.com", Username: "owner", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(ctx, owner))
	outsider := &domain.User{Email: "out@t.com", Username: "outsider", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(ctx, outsider))

	team := &domain.Team{Name: "Team", CreatedBy: owner.ID}
	require.NoError(t, teamRepo.Create(ctx, team))
	require.NoError(t, teamRepo.AddMember(ctx, &domain.TeamMember{
		UserID: owner.ID, TeamID: team.ID, Role: domain.RoleOwner,
	}))

	task := &domain.Task{
		Title: "Orphaned", Status: domain.StatusTodo,
		Priority: domain.PriorityMedium, TeamID: team.ID,
		CreatedBy: owner.ID, AssigneeID: &outsider.ID,
	}
	require.NoError(t, taskRepo.Create(ctx, task))

	orphaned, err := taskRepo.FindOrphaned(ctx)
	require.NoError(t, err)
	require.Len(t, orphaned, 1)
	assert.Equal(t, "Orphaned", orphaned[0].TaskTitle)
	assert.Equal(t, "outsider", orphaned[0].AssigneeName)
}

func TestCommentRepo_Integration(t *testing.T) {
	cleanTables(t)
	userRepo := mysqlrepo.NewUserRepository(testDB)
	teamRepo := mysqlrepo.NewTeamRepository(testDB)
	taskRepo := mysqlrepo.NewTaskRepository(testDB)
	commentRepo := mysqlrepo.NewCommentRepository(testDB)
	ctx := context.Background()

	user := &domain.User{Email: "u@t.com", Username: "u", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(ctx, user))
	team := &domain.Team{Name: "T", CreatedBy: user.ID}
	require.NoError(t, teamRepo.Create(ctx, team))
	task := &domain.Task{
		Title: "Task", Status: domain.StatusTodo,
		Priority: domain.PriorityMedium, TeamID: team.ID, CreatedBy: user.ID,
	}
	require.NoError(t, taskRepo.Create(ctx, task))

	comment := &domain.TaskComment{TaskID: task.ID, UserID: user.ID, Body: "Hello"}
	require.NoError(t, commentRepo.Create(ctx, comment))
	assert.NotZero(t, comment.ID)

	comments, err := commentRepo.ListByTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Equal(t, "Hello", comments[0].Body)
}
