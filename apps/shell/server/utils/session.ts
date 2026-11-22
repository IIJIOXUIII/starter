/**
 * Cookie-сессии BFF.
 *
 * - access (JWT, 15 мин)  — cookie `rt_access`, Path=/ : читается всеми API-роутами
 * - refresh (opaque, 12h/30d) — cookie `rt_refresh`, Path=/api/auth : доступен только auth-роутам
 *
 * Обе — httpOnly (XSS не может украсть сессию) + SameSite=Lax (CSRF)
 * + Secure на проде. Токены браузеру не отдаются — только { user }.
 */
import type { H3Event } from 'h3';
import type { AuthApi } from '@rosterra/api-client';

/** Токены сессии из контракта auth-service. */
type SessionTokens = AuthApi['SessionTokens'];

const ACCESS_COOKIE = 'rt_access';
const REFRESH_COOKIE = 'rt_refresh';

/** Secure-флаг: в dev на http://localhost Secure у части браузеров ломает cookie. */
const isProd = process.env.NODE_ENV === 'production';

const baseCookieOpts = {
  httpOnly: true,
  secure: isProd,
  sameSite: 'lax' as const,
};

/**
 * Записывает пару cookies сессии после register/login/refresh.
 *
 * @param event - событие h3.
 * @param tokens - токены из ответа auth-service.
 */
export function setAuthCookies(event: H3Event, tokens: SessionTokens): void {
  setCookie(event, ACCESS_COOKIE, tokens.access_token, {
    ...baseCookieOpts,
    path: '/',
    maxAge: tokens.expires_in,
  });
  setCookie(event, REFRESH_COOKIE, tokens.refresh_token, {
    ...baseCookieOpts,
    path: '/api/auth',
    maxAge: tokens.refresh_expires_in,
  });
}

/**
 * Удаляет cookies сессии (logout или истечение refresh).
 *
 * @param event - событие h3.
 */
export function clearAuthCookies(event: H3Event): void {
  deleteCookie(event, ACCESS_COOKIE, { path: '/' });
  deleteCookie(event, REFRESH_COOKIE, { path: '/api/auth' });
}

/**
 * Refresh-токен из cookie (только auth-роуты видят этот cookie).
 *
 * @param event - событие h3.
 * @returns значение cookie или пустая строка.
 */
export function readRefresh(event: H3Event): string {
  return getCookie(event, REFRESH_COOKIE) ?? '';
}

/**
 * Access-токен из cookie.
 *
 * @param event - событие h3.
 * @returns значение cookie или пустая строка.
 */
export function readAccess(event: H3Event): string {
  return getCookie(event, ACCESS_COOKIE) ?? '';
}

/**
 * Guard от CSRF для мутаций: если браузер прислал Origin, он обязан совпадать
 * с Host запроса (same-origin). SameSite=Lax + эта проверка — достаточный минимум.
 *
 * @param event - событие h3.
 * @returns true, если запрос допустим.
 */
export function isSameOrigin(event: H3Event): boolean {
  const origin = getHeader(event, 'origin');
  if (!origin) {
    return true; // same-origin запросы Origin могут не присылать
  }
  const host = getHeader(event, 'host');
  try {
    return new URL(origin).host === host;
  } catch {
    return false;
  }
}
