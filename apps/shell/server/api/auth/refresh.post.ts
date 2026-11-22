/**
 * POST /api/auth/refresh — ручная ротация сессии.
 *
 * Берёт refresh-cookie, вызывает auth-service (ротация + reuse detection),
 * обновляет cookies. UI обычно не вызывает сам: access обновляется прозрачно.
 */
import { authClient } from '../../utils/api';
import { clearSession, readRefresh, setSession } from '../../utils/session';
import { respondError } from '../../utils/proxy';

export default defineEventHandler(async (event) => {
  const refreshToken = readRefresh(event);
  if (!refreshToken) {
    clearSession(event);
    return respondError(event, 401, {
      code: 'unauthorized',
      message: 'authentication required',
    });
  }

  const { data, error, response } = await authClient.POST('/v1/refresh', {
    body: { refresh_token: refreshToken },
  });
  if (error || !data || response.status !== 200) {
    // Истёк/отозван/reuse — cookies бессмысленны, чистим.
    clearSession(event);
    return respondError(
      event,
      response.status,
      error ?? { code: 'internal', message: 'internal error' },
    );
  }

  setSession(event, data);
  return { user: data.user };
});
