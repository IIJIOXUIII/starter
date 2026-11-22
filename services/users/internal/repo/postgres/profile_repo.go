package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rosterra/services/users/internal/domain"
)

// ProfileRepo — domain.ProfileRepo на PostgreSQL.
type ProfileRepo struct {
	pool *pgxpool.Pool
}

// NewProfileRepo — fx-провайдер.
func NewProfileRepo(pool *pgxpool.Pool) *ProfileRepo {
	return &ProfileRepo{pool: pool}
}

const profileColumns = `user_id, display_name, avatar_url, notes, created_at, updated_at`

func scanProfile(row pgx.Row) (*domain.Profile, error) {
	var p domain.Profile
	err := row.Scan(&p.UserID, &p.DisplayName, &p.AvatarURL, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Create вставляет профиль; идемпотентно (ON CONFLICT DO NOTHING).
func (r *ProfileRepo) Create(ctx context.Context, p *domain.Profile) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO profiles (user_id, display_name, avatar_url)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO NOTHING`,
		p.UserID, p.DisplayName, p.AvatarURL)
	if err != nil {
		return fmt.Errorf("insert profile: %w", err)
	}
	return nil
}

// GetByID возвращает профиль или domain.ErrNotFound.
func (r *ProfileRepo) GetByID(ctx context.Context, userID uuid.UUID) (*domain.Profile, error) {
	p, err := scanProfile(r.pool.QueryRow(ctx, `SELECT `+profileColumns+` FROM profiles WHERE user_id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select profile: %w", err)
	}
	return p, nil
}

// ListByIDs возвращает найденные профили по списку id.
func (r *ProfileRepo) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Profile, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+profileColumns+` FROM profiles WHERE user_id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("select profiles: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Profile, 0, len(ids))
	for rows.Next() {
		var p domain.Profile
		if err := rows.Scan(&p.UserID, &p.DisplayName, &p.AvatarURL, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan profile: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows profiles: %w", err)
	}
	return items, nil
}

// Update применяет частичное обновление (nil = не менять, "" = очистить)
// и возвращает обновлённый профиль. Upsert: если события user.created ещё
// не долетели (лаг consumer'а), админская правка не падает с 404 —
// профиль создаётся из переданных полей с дефолтами.
func (r *ProfileRepo) Update(ctx context.Context, userID uuid.UUID, upd domain.UpdateProfile) (*domain.Profile, error) {
	p, err := scanProfile(r.pool.QueryRow(ctx, `
		INSERT INTO profiles AS p (user_id, display_name, avatar_url, notes)
		VALUES (
			$1,
			COALESCE($2, ''),
			CASE WHEN $3::text IS NULL OR $3::text = '' THEN NULL ELSE $3::text END,
			CASE WHEN $4::text IS NULL OR $4::text = '' THEN NULL ELSE $4::text END
		)
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = COALESCE($2, p.display_name),
			avatar_url   = CASE
				WHEN $3::text IS NULL THEN p.avatar_url
				WHEN $3::text = ''    THEN NULL
				ELSE $3::text
			END,
			notes        = CASE
				WHEN $4::text IS NULL THEN p.notes
				WHEN $4::text = ''    THEN NULL
				ELSE $4::text
			END,
			updated_at   = now()
		RETURNING `+profileColumns,
		userID, upd.DisplayName, upd.AvatarURL, upd.Notes))
	if err != nil {
		return nil, fmt.Errorf("upsert profile: %w", err)
	}
	return p, nil
}
