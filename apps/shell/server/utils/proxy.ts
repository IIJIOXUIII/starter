/**
 * Хелперы проксирования: error-envelope passthrough и прозрачный refresh.
 */
import type { H3Event } from 'h3';
import type { AuthApi } from '@rosterra/api-client';

import { authClient } from './api';
import { clearAuthCookies, readAccess, readRefresh, setAuthCookies } from './session';

/** Формат error-envelope всех сервисов платформы. */
export interface ErrorEnvelope {
  code: string;
  message: string;
  details?: { field: string; message: string }[];
}

/**
 * Ответ не-2xx: отдаёт статус и тело сервиса как есть (error-envelope passthrough).
 *
 * @param event - событие h3.
 * @param status - HTTP-статус из ответа сервиса.
 * @param body - error-envelope сервиса.
 * @returns тело ответа браузеру.
 */
export function respondError(event: H3Event, status: number, body: unknown): unknown {
  setResponseStatus(event, status);
  return body;
}

/**
 * Локальный 401 BFF (когда refresh тоже не удался).
 *
 * @param event - событие h3.
 * @returns envelope unauthorized.
 */
export function unauthorized(event: H3Event): unknown {
  return respondError(event, 401, {
    code: 'unauthorized',
    message: 'authentication required',
  } satisfies ErrorEnvelope);
}

/** Результат прозрачной ротации. */
export interface RefreshResult {
  ok: boolean;
  /** Новая сессия (user + токены) при успехе. */
  tokens?: AuthApi['SessionTokens'];
}

/**
 * Прозрачная ротация сессии: refresh-cookie → auth-service → новые cookies.
 * При неудаче cookies очищаются (refresh истёк/отозван — повторять бессмысленно).
 *
 * @param event - событие h3.
 * @returns ok=true и новые токены, если ротация удалась.
 */
export async function refreshSession(event: H3Event): Promise<RefreshResult> {
  const refreshToken = readRefresh(event);
  if (!refreshToken) {
    return { ok: false };
  }
  const { data, error, response } = await authClient.POST('/v1/refresh', {
    body: { refresh_token: refreshToken },
  });
  if (error || !data || response.status !== 200) {
    clearAuthCookies(event);
    return { ok: false };
  }
  setAuthCookies(event, data);
  return { ok: true, tokens: data };
}

/**
 * Authorization-заголовок из access-cookie, если она есть.
 *
 * @param event - событие h3.
 * @returns значение `Bearer <token>` или undefined.
 */
export function bearerFromCookie(event: H3Event): string | undefined {
  const access = readAccess(event);
  return access ? `Bearer ${access}` : undefined;
}
