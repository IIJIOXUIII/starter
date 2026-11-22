/**
 * GET /api/admin/users/:id — карточка пользователя: user (auth) + профиль (users).
 * Доступ — только admin.
 */
import { authorizedUsersClient } from '../../../utils/api';
import { callAuth, requireAdmin } from '../../../utils/admin';
import { respondError } from '../../../utils/proxy';

export default defineEventHandler(async (event) => {
  const guard = await requireAdmin(event);
  if (!guard.ok) {
    return respondError(event, guard.status, guard.body);
  }

  const id = getRouterParam(event, 'id') ?? '';
  const user = await callAuth(event, (client) =>
    client.GET('/v1/admin/users/{id}', { params: { path: { id } } }),
  );
  if (!user.ok) {
    return respondError(event, user.status, user.body);
  }

  const { data: profile, response } = await authorizedUsersClient(guard.token).GET(
    '/v1/profiles/{user_id}',
    {
      params: { path: { user_id: id } },
    },
  );
  // 404 профиля — норма (событие ещё не дошло); остальные ошибки — passthrough.
  if (response && !response.ok && response.status !== 404) {
    return respondError(event, response.status, {
      code: 'internal',
      message: 'profiles service unavailable',
    });
  }

  return { user: user.data, profile: profile ?? null };
});
