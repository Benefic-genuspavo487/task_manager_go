package usecase_test

import (
	"context"

	"github.com/loks1k192/task-manager/internal/domain"
)

type mockUserRepo struct {
	users  map[int64]*domain.User
	byEmail map[string]*domain.User
	nextID int64
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[int64]*domain.User),
		byEmail: make(map[string]*domain.User),
		nextID:  1,
	}
}

func (m *mockUserRepo) Create(_ context.Context, user *domain.User) error {
	if _, exists := m.byEmail[user.Email]; exists {
		return domain.ErrConflict
	}
	user.ID = m.nextID
	m.nextID++
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

type mockTeamRepo struct {
	teams      map[int64]*domain.Team
	members    map[int64]map[int64]*domain.TeamMember // teamID -> userID -> member
	nextID     int64
	nextMemberID int64
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{
		teams:   make(map[int64]*domain.Team),
		members: make(map[int64]map[int64]*domain.TeamMember),
		nextID:  1,
		nextMemberID: 1,
	}
}

func (m *mockTeamRepo) Create(_ context.Context, team *domain.Team) error {
	team.ID = m.nextID
	m.nextID++
	m.teams[team.ID] = team
	return nil
}

func (m *mockTeamRepo) GetByID(_ context.Context, id int64) (*domain.Team, error) {
	t, ok := m.teams[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (m *mockTeamRepo) ListByUser(_ context.Context, userID int64) ([]domain.Team, error) {
	var result []domain.Team
	for teamID, members := range m.members {
		if _, ok := members[userID]; ok {
			if t, ok := m.teams[teamID]; ok {
				result = append(result, *t)
			}
		}
	}
	return result, nil
}

func (m *mockTeamRepo) AddMember(_ context.Context, member *domain.TeamMember) error {
	if m.members[member.TeamID] == nil {
		m.members[member.TeamID] = make(map[int64]*domain.TeamMember)
	}
	if _, exists := m.members[member.TeamID][member.UserID]; exists {
		return domain.ErrConflict
	}
	member.ID = m.nextMemberID
	m.nextMemberID++
	m.members[member.TeamID][member.UserID] = member
	return nil
}

func (m *mockTeamRepo) GetMember(_ context.Context, userID, teamID int64) (*domain.TeamMember, error) {
	if members, ok := m.members[teamID]; ok {
		if member, ok := members[userID]; ok {
			return member, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockTeamRepo) IsMember(_ context.Context, userID, teamID int64) (bool, error) {
	if members, ok := m.members[teamID]; ok {
		_, exists := members[userID]
		return exists, nil
	}
	return false, nil
}

func (m *mockTeamRepo) GetStats(_ context.Context) ([]domain.TeamStats, error) {
	return nil, nil
}

func (m *mockTeamRepo) GetTopCreators(_ context.Context, _, _ int) ([]domain.TopCreator, error) {
	return nil, nil
}

type mockTaskRepo struct {
	tasks   map[int64]*domain.Task
	history map[int64][]domain.TaskHistory
	nextID  int64
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{
		tasks:   make(map[int64]*domain.Task),
		history: make(map[int64][]domain.TaskHistory),
		nextID:  1,
	}
}

func (m *mockTaskRepo) Create(_ context.Context, task *domain.Task) error {
	task.ID = m.nextID
	m.nextID++
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) GetByID(_ context.Context, id int64) (*domain.Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (m *mockTaskRepo) Update(_ context.Context, task *domain.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) List(_ context.Context, _ domain.TaskFilter) ([]domain.Task, error) {
	var result []domain.Task
	for _, t := range m.tasks {
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockTaskRepo) AddHistory(_ context.Context, h *domain.TaskHistory) error {
	m.history[h.TaskID] = append(m.history[h.TaskID], *h)
	return nil
}

func (m *mockTaskRepo) GetHistory(_ context.Context, taskID int64) ([]domain.TaskHistory, error) {
	return m.history[taskID], nil
}

func (m *mockTaskRepo) FindOrphaned(_ context.Context) ([]domain.OrphanedTask, error) {
	return nil, nil
}

type mockCommentRepo struct {
	comments map[int64][]domain.TaskComment
	nextID   int64
}

func newMockCommentRepo() *mockCommentRepo {
	return &mockCommentRepo{
		comments: make(map[int64][]domain.TaskComment),
		nextID:   1,
	}
}

func (m *mockCommentRepo) Create(_ context.Context, c *domain.TaskComment) error {
	c.ID = m.nextID
	m.nextID++
	m.comments[c.TaskID] = append(m.comments[c.TaskID], *c)
	return nil
}

func (m *mockCommentRepo) ListByTask(_ context.Context, taskID int64) ([]domain.TaskComment, error) {
	return m.comments[taskID], nil
}

type mockCacheRepo struct {
	store map[string][]byte
}

func newMockCacheRepo() *mockCacheRepo {
	return &mockCacheRepo{store: make(map[string][]byte)}
}

func (m *mockCacheRepo) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := m.store[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (m *mockCacheRepo) Set(_ context.Context, key string, value []byte, _ int) error {
	m.store[key] = value
	return nil
}

func (m *mockCacheRepo) Delete(_ context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func (m *mockCacheRepo) DeleteByPrefix(_ context.Context, prefix string) error {
	for k := range m.store {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(m.store, k)
		}
	}
	return nil
}

type mockEmailService struct {
	sent []string
}

func (m *mockEmailService) SendInvite(_ context.Context, email, _ string) error {
	m.sent = append(m.sent, email)
	return nil
}
