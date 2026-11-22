package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rosterra/services/auth/internal/domain"
)

// UserRepo — PostgreSQL-реализация domain.UserRepo.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo конструирует репозиторий пользователей.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

const userCols = `
	u.id, u.email, u.name, u.status, u.email_verified, u.created_at, u.updated_at, u.password_hash,
	COALESCE((SELECT array_agg(r.role ORDER BY r.role) FROM user_roles r WHERE r.user_id = u.id), '{}') AS roles`

const userSelect = `SELECT ` + userCols + ` FROM users u WHERE `

// Create вставляет пользователя, его роли и событие outbox в одной транзакции.
func (r *UserRepo) Create(ctx context.Context, u *domain.User, passwordHash string, event *domain.OutboxEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback после успешного Commit безопасен

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, status, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.ID, u.Email, passwordHash, u.Name, string(u.Status), u.EmailVerified, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	for _, role := range u.Roles {
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, u.ID, string(role)); err != nil {
			return fmt.Errorf("insert role %q: %w", role, err)
		}
	}

	if event != nil {
		if _, err := tx.Exec(ctx,
			`INSERT INTO outbox (id, type, payload, created_at) VALUES ($1, $2, $3, $4)`,
			event.ID, event.Type, event.Payload, event.CreatedAt); err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetByEmail возвращает пользователя по email (citext — без учёта регистра).
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, userSelect+`u.email = $1`, email)
	return scanUser(row)
}

// GetByID возвращает пользователя по id.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, userSelect+`u.id = $1`, id)
	return scanUser(row)
}

// UpdateStatus меняет статус и пишет событие в outbox одной транзакцией.
func (r *UserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus, event *domain.OutboxEvent) (*domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx,
		`UPDATE users SET status = $2, updated_at = now() WHERE id = $1`, id, string(status))
	if err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrNotFound
	}
	if event != nil {
		if _, err := tx.Exec(ctx,
			`INSERT INTO outbox (id, type, payload, created_at) VALUES ($1, $2, $3, $4)`,
			event.ID, event.Type, event.Payload, event.CreatedAt); err != nil {
			return nil, fmt.Errorf("insert outbox: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, id)
}

// SetRoles заменяет набор ролей и пишет событие в outbox одной транзакцией.
func (r *UserRepo) SetRoles(ctx context.Context, id uuid.UUID, roles []domain.Role, event *domain.OutboxEvent) (*domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `UPDATE users SET updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("touch user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, id); err != nil {
		return nil, fmt.Errorf("delete roles: %w", err)
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, id, string(role)); err != nil {
			return nil, fmt.Errorf("insert role %q: %w", role, err)
		}
	}
	if event != nil {
		if _, err := tx.Exec(ctx,
			`INSERT INTO outbox (id, type, payload, created_at) VALUES ($1, $2, $3, $4)`,
			event.ID, event.Type, event.Payload, event.CreatedAt); err != nil {
			return nil, fmt.Errorf("insert outbox: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, id)
}

// List возвращает страницу пользователей по фильтру (считая total).
func (r *UserRepo) List(ctx context.Context, f domain.ListFilter) ([]domain.User, int, error) {
	where := " WHERE true"
	args := make([]any, 0, 4)

	if f.Query != nil && *f.Query != "" {
		args = append(args, "%"+*f.Query+"%")
		where += fmt.Sprintf(" AND u.email ILIKE $%d", len(args))
	}
	if f.Status != nil {
		args = append(args, string(*f.Status))
		where += fmt.Sprintf(" AND u.status = $%d", len(args))
	}
	if f.Role != nil {
		args = append(args, string(*f.Role))
		where += fmt.Sprintf(
			" AND EXISTS (SELECT 1 FROM user_roles rr WHERE rr.user_id = u.id AND rr.role = $%d)", len(args))
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM users u`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}

	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)
	limitArg, offsetArg := len(args)-1, len(args)

	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(
			`SELECT %s FROM users u%s ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d`,
			userCols, where, limitArg, offsetArg),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("select list: %w", err)
	}
	defer rows.Close()

	items := make([]domain.User, 0, f.PerPage)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}
	return items, total, nil
}

// scanUser читает пользователя (со столбцом ролей array_agg) из строки.
func scanUser(row pgx.Row) (*domain.User, error) {
	var (
		u      domain.User
		email  string
		name   *string
		status string
		roles  []string
	)
	if err := row.Scan(&u.ID, &email, &name, &status, &u.EmailVerified,
		&u.CreatedAt, &u.UpdatedAt, &u.PasswordHash, &roles); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Email, u.Name, u.Status = email, name, domain.UserStatus(status)
	u.Roles = make([]domain.Role, 0, len(roles))
	for _, r := range roles {
		u.Roles = append(u.Roles, domain.Role(r))
	}
	return &u, nil
}
