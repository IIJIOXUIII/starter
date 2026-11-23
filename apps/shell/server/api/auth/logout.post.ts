/**
 * POST /api/auth/logout — выход.
 *
 * Отзывает refresh-сессию в auth-service (если cookie была) и всегда
 * очищает cookies: logout не должен ломаться об уже протухшую сессию.
 */
import { authClient, traceHeaders } from '../../utils/api';
import { bearerFromCookie, respondError } from '../../utils/proxy';
import { clearAuthCookies, isSameOrigin, readRefresh } from '../../utils/session';

export default defineEventHandler(async (event) => {
  if (!isSameOrigin(event)) {
    return respondError(event, 403, { code: 'forbidden', message: 'cross-origin request blocked' });
  }

  const refreshToken = readRefresh(event);
  const authorization = bearerFromCookie(event);

  if (refreshToken && authorization) {
    // Результат не критичен: даже если сессия уже отозвана — cookies чистим.
    await authClient.POST('/v1/logout', {
      headers: { Authorization: authorization, ...traceHeaders(event) },
      body: { refresh_token: refreshToken },
    });
  }

  clearAuthCookies(event);
  setResponseStatus(event, 204);
  return null;
});
