package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
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

type ListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
	Filter    string
}

type ListResult struct {
	Items    []UserRecord
	Total    int
	Page     int
	PageSize int
}

type Repository interface {
	ListUsers(ctx context.Context, query ListQuery) (ListResult, error)
	CreateUser(ctx context.Context, username, passwordHash string, isActive bool) (UserRecord, error)
	SetUserActive(ctx context.Context, userID int64, isActive bool) error
	SetUsername(ctx context.Context, userID int64, username string) error
	SetPassword(ctx context.Context, userID int64, passwordHash string) error
}

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func normalizeListQuery(query ListQuery) ListQuery {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 25
	}
	sortField := strings.TrimSpace(query.SortField)
	switch sortField {
	case "id", "username", "created_at", "updated_at", "is_active":
	default:
		sortField = "id"
	}
	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "desc" {
		sortOrder = "asc"
	}

	return ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    strings.TrimSpace(query.Filter),
	}
}

func (r *SQLRepository) ListUsers(ctx context.Context, query ListQuery) (ListResult, error) {
	q := normalizeListQuery(query)
	offset := (q.Page - 1) * q.PageSize
	filterValue := "%"
	hasFilter := q.Filter != ""
	if hasFilter {
		filterValue = "%" + q.Filter + "%"
	}

	total := 0
	countArgs := []any{}
	countQuery := `SELECT COUNT(*) FROM users`
	if hasFilter {
		countQuery += ` WHERE username ILIKE $1`
		countArgs = append(countArgs, filterValue)
	}
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return ListResult{}, fmt.Errorf("count users: %w", err)
	}

	items := []UserRecord{}
	selectArgs := []any{q.PageSize, offset}
	selectQuery := `
		SELECT id, username, is_active, created_at, updated_at
		FROM users
	`
	if hasFilter {
		selectQuery += ` WHERE username ILIKE $3`
		selectArgs = append(selectArgs, filterValue)
	}
	selectQuery += fmt.Sprintf(" ORDER BY %s %s LIMIT $1 OFFSET $2", q.SortField, strings.ToUpper(q.SortOrder))

	err := r.db.SelectContext(ctx, &items, selectQuery, selectArgs...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list users: %w", err)
	}
	for i := range items {
		items[i].Roles = []string{}
	}

	return ListResult{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
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

func (r *SQLRepository) SetUsername(ctx context.Context, userID int64, username string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET username = $2, updated_at = NOW() WHERE id = $1`, userID, strings.TrimSpace(username))
	if err != nil {
		return fmt.Errorf("set username: %w", err)
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

func (s *Service) ListUsers(ctx context.Context, query ListQuery) (ListResult, error) {
	users, err := s.repo.ListUsers(ctx, query)
	if err != nil {
		return ListResult{}, err
	}
	for i := range users.Items {
		roles, roleErr := s.rbacService.RoleNamesForUser(ctx, users.Items[i].ID)
		if roleErr != nil {
			return ListResult{}, roleErr
		}
		users.Items[i].Roles = roles
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
	if strings.TrimSpace(in.Username) == "" || strings.TrimSpace(in.Password) == "" {
		return UserRecord{}, apperror.Validation("username and password are required")
	}
	hash, err := auth.HashPassword(in.Password, s.passwordHashCost)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserRecord{}, apperror.Validation("one or more roles are invalid")
		}
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
	Username  *string
	Roles     *[]string
	IsActive  *bool
	ActorID   *int64
	ResetRBAC bool
}

func (s *Service) UpdateUser(ctx context.Context, in UpdateInput) error {
	metadata := map[string]any{}

	if in.Username != nil {
		username := strings.TrimSpace(*in.Username)
		if username != "" {
			if err := s.repo.SetUsername(ctx, in.UserID, username); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return apperror.Validation("user not found")
				}
				return err
			}
			metadata["username"] = username
		}
	}

	if in.IsActive != nil {
		if err := s.repo.SetUserActive(ctx, in.UserID, *in.IsActive); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apperror.Validation("user not found")
			}
			return err
		}
		s.logAudit(ctx, audit.Event{Action: "users.set_active", ActorUserID: in.ActorID, EntityType: "user", EntityID: strPtr(fmt.Sprintf("%d", in.UserID)), Metadata: map[string]any{"is_active": *in.IsActive}})
	}

	if in.Roles != nil {
		cleanRoles := make([]string, 0, len(*in.Roles))
		for _, role := range *in.Roles {
			if strings.TrimSpace(role) == "" {
				continue
			}
			cleanRoles = append(cleanRoles, strings.TrimSpace(role))
		}
		if err := s.rbacService.SetUserRoles(ctx, in.UserID, cleanRoles); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apperror.Validation("one or more roles are invalid")
			}
			return err
		}
		metadata["roles"] = cleanRoles
	}

	if len(metadata) > 0 {
		s.logAudit(ctx, audit.Event{
			Action:      "users.update",
			ActorUserID: in.ActorID,
			EntityType:  "user",
			EntityID:    strPtr(fmt.Sprintf("%d", in.UserID)),
			Metadata:    metadata,
		})
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
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.Validation("user not found")
		}
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
