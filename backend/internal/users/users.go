package users

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	"github.com/jmoiron/sqlx"
)

type UserRecord struct {
	ID        int64     `db:"id" json:"id"`
	Username  string    `db:"username" json:"username"`
	IsActive  bool      `db:"is_active" json:"isActive"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
	Roles     []string  `json:"roles"`
}

type Repository interface {
	ListUsers(ctx context.Context) ([]UserRecord, error)
	CreateUser(ctx context.Context, username, passwordHash string, isActive bool) (UserRecord, error)
	SetUserActive(ctx context.Context, userID int64, isActive bool) error
	SetPassword(ctx context.Context, userID int64, passwordHash string) error
}

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) ListUsers(ctx context.Context) ([]UserRecord, error) {
	items := []UserRecord{}
	err := r.db.SelectContext(ctx, &items, `
		SELECT id, username, is_active, created_at, updated_at
		FROM users
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	for i := range items {
		items[i].Roles = []string{}
	}
	return items, nil
}

func (r *SQLRepository) CreateUser(ctx context.Context, username, passwordHash string, isActive bool) (UserRecord, error) {
	var record UserRecord
	err := r.db.GetContext(ctx, &record, `
		INSERT INTO users (username, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, username, is_active, created_at, updated_at
	`, strings.TrimSpace(username), passwordHash, isActive)
	if err != nil {
		return UserRecord{}, fmt.Errorf("create user: %w", err)
	}
	record.Roles = []string{}
	return record, nil
}

func (r *SQLRepository) SetUserActive(ctx context.Context, userID int64, isActive bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET is_active = $2, updated_at = NOW() WHERE id = $1`, userID, isActive)
	if err != nil {
		return fmt.Errorf("set user active: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLRepository) SetPassword(ctx context.Context, userID int64, passwordHash string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type Service struct {
	repo             Repository
	rbacService      *rbac.Service
	auditService     *audit.Service
	passwordHashCost int
}

func NewService(repo Repository, rbacService *rbac.Service, auditService *audit.Service, passwordHashCost int) *Service {
	return &Service{repo: repo, rbacService: rbacService, auditService: auditService, passwordHashCost: passwordHashCost}
}

func (s *Service) ListUsers(ctx context.Context) ([]UserRecord, error) {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		roles, roleErr := s.rbacService.RoleNamesForUser(ctx, users[i].ID)
		if roleErr != nil {
			return nil, roleErr
		}
		users[i].Roles = roles
	}
	return users, nil
}

type CreateInput struct {
	Username string
	Password string
	IsActive bool
	Roles    []string
	ActorID  *int64
}

func (s *Service) CreateUser(ctx context.Context, in CreateInput) (UserRecord, error) {
	hash, err := auth.HashPassword(in.Password, s.passwordHashCost)
	if err != nil {
		return UserRecord{}, err
	}
	created, err := s.repo.CreateUser(ctx, in.Username, hash, in.IsActive)
	if err != nil {
		return UserRecord{}, err
	}
	for _, role := range in.Roles {
		if strings.TrimSpace(role) == "" {
			continue
		}
		if assignErr := s.rbacService.AssignRoleToUser(ctx, created.ID, role); assignErr != nil {
			return UserRecord{}, assignErr
		}
	}
	r, err := s.rbacService.RoleNamesForUser(ctx, created.ID)
	if err != nil {
		return UserRecord{}, err
	}
	created.Roles = r
	s.logAudit(ctx, audit.Event{Action: "users.create", ActorUserID: in.ActorID, EntityType: "user", EntityID: strPtr(created.Username), Metadata: map[string]any{"user_id": created.ID, "roles": created.Roles}})
	return created, nil
}

type UpdateInput struct {
	UserID    int64
	Roles     []string
	IsActive  *bool
	ActorID   *int64
	ResetRBAC bool
}

func (s *Service) UpdateUser(ctx context.Context, in UpdateInput) error {
	if in.IsActive != nil {
		if err := s.repo.SetUserActive(ctx, in.UserID, *in.IsActive); err != nil {
			return err
		}
		s.logAudit(ctx, audit.Event{Action: "users.set_active", ActorUserID: in.ActorID, EntityType: "user", EntityID: strPtr(fmt.Sprintf("%d", in.UserID)), Metadata: map[string]any{"is_active": *in.IsActive}})
	}
	for _, role := range in.Roles {
		if strings.TrimSpace(role) == "" {
			continue
		}
		if err := s.rbacService.AssignRoleToUser(ctx, in.UserID, role); err != nil {
			return err
		}
	}
	if len(in.Roles) > 0 {
		s.logAudit(ctx, audit.Event{Action: "users.update", ActorUserID: in.ActorID, EntityType: "user", EntityID: strPtr(fmt.Sprintf("%d", in.UserID)), Metadata: map[string]any{"roles": in.Roles}})
	}
	s.rbacService.InvalidateUser(in.UserID)
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, actorID *int64, userID int64, newPassword string) error {
	hash, err := auth.HashPassword(newPassword, s.passwordHashCost)
	if err != nil {
		return err
	}
	if err := s.repo.SetPassword(ctx, userID, hash); err != nil {
		return err
	}
	s.logAudit(ctx, audit.Event{Action: "users.reset_password", ActorUserID: actorID, EntityType: "user", EntityID: strPtr(fmt.Sprintf("%d", userID))})
	return nil
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func strPtr(v string) *string {
	return &v
}
