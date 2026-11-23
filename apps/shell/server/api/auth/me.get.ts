/**
 * GET /api/auth/me — текущий пользователь.
 *
 * 1) Пробует access-cookie.
 * 2) Если его нет или он истёк (401) — прозрачный refresh и повтор.
 * 3) Refresh тоже не удался → 401 + чистые cookies.
 */
import { authorizedAuthClient } from '../../utils/api';
import { refreshSession, respondError } from '../../utils/proxy';
import { readAccess } from '../../utils/session';

export default defineEventHandler(async (event) => {
  const access = readAccess(event);

  if (access) {
    const { data, response } = await authorizedAuthClient(access, event).GET('/v1/me');
    if (!response || response.status !== 401) {
      if (data) {
        return { user: data };
      }
      return respondError(event, response?.status ?? 500, {
        code: 'internal',
        message: 'internal error',
      });
    }
  }

  // Access отсутствует/истёк — прозрачная ротация.
  const refreshed = await refreshSession(event);
  if (!refreshed.ok || !refreshed.tokens) {
    return respondError(event, 401, { code: 'unauthorized', message: 'authentication required' });
  }

  const { data, error, response } = await authorizedAuthClient(
    refreshed.tokens.access_token,
    event,
  ).GET('/v1/me');
  if (error || !data) {
    return respondError(
      event,
      response.status,
      error ?? { code: 'internal', message: 'internal error' },
    );
  }
  return { user: data };
});
