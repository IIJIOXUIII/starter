package service

// Юнит-тесты сервиса профилей (фейки вместо БД).

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/rosterra/services/users/internal/domain"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestService() (*Service, *fakeProfileRepo, *fakeOutboxRepo, *fakeProcessedRepo) {
	log := discardLogger()
	profiles := newFakeProfileRepo()
	outbox := &fakeOutboxRepo{}
	processed := newFakeProcessedRepo()
	return New(profiles, outbox, processed, log), profiles, outbox, processed
}

func TestCreateProfileIdempotent(t *testing.T) {
	svc, profiles, _, _ := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	first, err := svc.CreateProfile(ctx, userID, "Anna", "")
	require.NoError(t, err)
	require.Equal(t, "Anna", first.DisplayName)

	// Повторное создание (redelivery/повторный вызов) — no-op, имя не затирается.
	second, err := svc.CreateProfile(ctx, userID, "", "")
	require.NoError(t, err)
	require.Equal(t, "Anna", second.DisplayName)

	stored, err := profiles.GetByID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, "Anna", stored.DisplayName)
	require.Nil(t, stored.AvatarURL)
}

func TestCreateProfileValidation(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.CreateProfile(ctx, uuid.New(), longString(maxDisplayName+1), "")
	var ve *ValidateError
	require.True(t, errors.As(err, &ve), "want ValidateError, got %v", err)

	_, err = svc.CreateProfile(ctx, uuid.New(), "ok", longString(maxAvatarURL+1))
	require.True(t, errors.As(err, &ve))
}

func TestListProfilesLimit(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := context.Background()

	ids := make([]uuid.UUID, maxBatchIDs+1)
	for i := range ids {
		ids[i] = uuid.New()
	}
	_, err := svc.ListProfiles(ctx, ids)
	var ve *ValidateError
	require.True(t, errors.As(err, &ve))

	// Пустой список — валидно, возвращает пустой результат.
	items, err := svc.ListProfiles(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, items)

	// Найденные id возвращаются, отсутствующие — просто пропадают.
	known := uuid.New()
	_, err = svc.CreateProfile(ctx, known, "Known", "")
	require.NoError(t, err)
	items, err = svc.ListProfiles(ctx, []uuid.UUID{known, uuid.New()})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, known, items[0].UserID)
}

func TestUpdateProfilePartialAndClear(t *testing.T) {
	svc, profiles, outbox, _ := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	avatar := "https://example.com/a.png"
	notes := "vip"
	_, err := svc.CreateProfile(ctx, userID, "Anna", avatar)
	require.NoError(t, err)
	_, err = profiles.Update(ctx, userID, domain.UpdateProfile{Notes: &notes})
	require.NoError(t, err)

	// Меняем только имя: avatar/notes сохраняются.
	updated, err := svc.UpdateProfile(ctx, userID, domain.UpdateProfile{DisplayName: ptr("Bob")})
	require.NoError(t, err)
	require.Equal(t, "Bob", updated.DisplayName)
	require.NotNil(t, updated.AvatarURL)
	require.Equal(t, avatar, *updated.AvatarURL)
	require.NotNil(t, updated.Notes)

	// Пустая строка = очистить.
	updated, err = svc.UpdateProfile(ctx, userID, domain.UpdateProfile{
		AvatarURL: ptr(""),
		Notes:     ptr(""),
	})
	require.NoError(t, err)
	require.Nil(t, updated.AvatarURL)
	require.Nil(t, updated.Notes)

	stored, err := profiles.GetByID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, "Bob", stored.DisplayName)
	require.Nil(t, stored.AvatarURL)
	require.Nil(t, stored.Notes)

	// Каждый UpdateProfile кладёт profile.updated в outbox (два апдейта выше).
	events := outbox.published()
	require.Len(t, events, 2)
	for _, evt := range events {
		require.Equal(t, EventTypeProfileUpdated, evt.Type)
	}
}

func TestUpdateProfileUpsertOnMissing(t *testing.T) {
	// Правка профиля до прибытия события user.created не падает:
	// Update — upsert (само-исцеление лага событий).
	svc, profiles, _, _ := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	name := "EarlyEdit"
	updated, err := svc.UpdateProfile(ctx, userID, domain.UpdateProfile{DisplayName: &name})
	require.NoError(t, err)
	require.Equal(t, name, updated.DisplayName)

	stored, err := profiles.GetByID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, name, stored.DisplayName)
	require.Nil(t, stored.AvatarURL)
}

func TestUpdateProfileValidation(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := context.Background()
	userID := uuid.New()
	_, err := svc.CreateProfile(ctx, userID, "Anna", "")
	require.NoError(t, err)

	long := longString(maxDisplayName + 1)
	_, err = svc.UpdateProfile(ctx, userID, domain.UpdateProfile{DisplayName: &long})
	var ve *ValidateError
	require.True(t, errors.As(err, &ve))
}

func TestHandleUserCreatedDedup(t *testing.T) {
	svc, profiles, _, processed := newTestService()
	ctx := context.Background()
	userID := uuid.New()
	eventID := uuid.New()
	name := "FromEvent"

	require.NoError(t, svc.HandleUserCreated(ctx, eventID, userID, &name))
	stored, err := profiles.GetByID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, name, stored.DisplayName)

	// Повторная доставка того же события — skip.
	require.NoError(t, svc.HandleUserCreated(ctx, eventID, userID, &name))
	stored, err = profiles.GetByID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, name, stored.DisplayName)

	// Новое событие по тому же пользователю — CreateProfile идемпотентен.
	require.NoError(t, svc.HandleUserCreated(ctx, uuid.New(), userID, &name))
	stored, err = profiles.GetByID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, name, stored.DisplayName)
	require.Len(t, processed.seen, 2)
}

// ptr — хелпер указателя на строку.
func ptr(v string) *string {
	return &v
}
