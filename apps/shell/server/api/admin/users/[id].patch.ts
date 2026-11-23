/**
 * PATCH /api/admin/users/:id — сохранение карточки из админки.
 *
 * Тело (все части необязательны):
 *   { status?: 'active'|'blocked', reason?: string,
 *     roles?: ('user'|'admin')[],
 *     profile?: { display_name?: string, avatar_url?: string|null, notes?: string|null } }
 *
 * Статус/роли — auth-service; профиль — users-service (notes — только admin).
 * Семантика профиля: '' — очистить поле, отсутствие ключа — не менять.
 * Доступ — только admin.
 */
import { authorizedUsersClient } from '../../../utils/api';
import { callAuth, requireAdmin } from '../../../utils/admin';
import { respondError } from '../../../utils/proxy';

interface PatchBody {
  status?: string;
  reason?: string;
  roles?: string[];
  profile?: {
    display_name?: string;
    avatar_url?: string | null;
    notes?: string | null;
  };
}

export default defineEventHandler(async (event) => {
  const guard = await requireAdmin(event);
  if (!guard.ok) {
    return respondError(event, guard.status, guard.body);
  }

  const id = getRouterParam(event, 'id') ?? '';
  const body = (await readBody<PatchBody>(event).catch(() => null)) as PatchBody | null;
  if (!body || typeof body !== 'object') {
    return respondError(event, 400, {
      code: 'validation_failed',
      message: 'request body required',
    });
  }

  let user = null;

  // 1) Статус (auth-service): блокировка отзывает все сессии пользователя.
  if (body.status !== undefined) {
    const res = await callAuth(event, (client) =>
      client.PATCH('/v1/admin/users/{id}/status', {
        params: { path: { id } },
        body: { status: body.status as 'active' | 'blocked', reason: body.reason || undefined },
      }),
    );
    if (!res.ok) {
      return respondError(event, res.status, res.body);
    }
    user = res.data;
  }

  // 2) Роли (auth-service).
  if (body.roles !== undefined) {
    const res = await callAuth(event, (client) =>
      client.PUT('/v1/admin/users/{id}/roles', {
        params: { path: { id } },
        body: { roles: body.roles as ('user' | 'admin')[] },
      }),
    );
    if (!res.ok) {
      return respondError(event, res.status, res.body);
    }
    user = res.data;
  }

  // 3) Профиль (users-service). '' в avatar_url/notes = очистить поле.
  let profile = null;
  if (body.profile !== undefined) {
    const { data, error, response } = await authorizedUsersClient(guard.token, event).PATCH(
      '/v1/profiles/{user_id}',
      {
        params: { path: { user_id: id } },
        body: {
          display_name: body.profile.display_name ?? undefined,
          avatar_url: body.profile.avatar_url ?? undefined,
          notes: body.profile.notes ?? undefined,
        },
      },
    );
    if (error || !response?.ok) {
      return respondError(
        event,
        response?.status ?? 500,
        error ?? { code: 'internal', message: 'internal error' },
      );
    }
    profile = data;
  }

  return { user, profile };
});
