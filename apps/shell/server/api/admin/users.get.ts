/**
 * GET /api/admin/users — агрегированный admin-список: пользователи auth-service
 * + профили users-service. Доступ — только admin (guard с прозрачным refresh).
 */
import { authorizedUsersClient } from '../../utils/api';
import { callAuth, mergeProfiles, requireAdmin } from '../../utils/admin';
import { respondError } from '../../utils/proxy';

export default defineEventHandler(async (event) => {
  const guard = await requireAdmin(event);
  if (!guard.ok) {
    return respondError(event, guard.status, guard.body);
  }

  const query = getQuery(event);
  const listQuery = {
    query: typeof query.query === 'string' && query.query !== '' ? query.query : undefined,
    status: query.status === 'active' || query.status === 'blocked' ? query.status : undefined,
    role: query.role === 'user' || query.role === 'admin' ? query.role : undefined,
    page: positiveInt(query.page, 1),
    per_page: positiveInt(query.per_page, 20),
  };

  const page = await callAuth(event, (client) =>
    client.GET('/v1/admin/users', { query: listQuery }),
  );
  if (!page.ok) {
    return respondError(event, page.status, page.body);
  }

  const users = page.data.items;
  const ids = users.map((u) => u.id).join(',');
  let profiles = null;
  if (ids !== '') {
    const { data, error, response } = await authorizedUsersClient(guard.token).GET('/v1/profiles', {
      params: { query: { ids } },
    });
    if (error || !response?.ok) {
      return respondError(event, response?.status ?? 500, {
        code: 'internal',
        message: 'profiles service unavailable',
      });
    }
    profiles = data.items;
  }

  return {
    items: mergeProfiles(users, profiles),
    total: page.data.total,
    page: page.data.page,
    per_page: page.data.per_page,
  };
});

/**
 * Приводит параметр запроса к положительному int.
 *
 * @param raw - сырое значение query.
 * @param fallback - значение по умолчанию.
 * @returns число или fallback.
 */
function positiveInt(raw: unknown, fallback: number): number {
  const n = Number(raw);
  return Number.isInteger(n) && n >= 1 ? n : fallback;
}
