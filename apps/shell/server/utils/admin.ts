/**
 * Админ-слой BFF: guard (role admin + прозрачный refresh) и агрегация
 * данных auth (список пользователей) с профилями users-service.
 */
import type { H3Event } from 'h3';
import type { AuthApi, UsersApi } from '@rosterra/api-client';

import { authorizedAuthClient } from './api';
import { refreshSession } from './proxy';
import { readAccess } from './session';

/** Пользователь auth-service (контракт admin-эндпоинтов). */
export type AdminUser = AuthApi['User'];
/** Профиль users-service. */
export type UserProfile = UsersApi['Profile'];

/** Строка агрегированного списка админки. */
export interface AdminUserRow {
  user: AdminUser;
  /** null — профиль ещё не создан (событие user.created в полёте). */
  profile: UserProfile | null;
}

/** Общий тип вызова openapi-fetch. */
export interface ApiCall<T> {
  data?: T;
  error?: unknown;
  response?: Response;
}

/** Клиент auth-service с bearer (возвращается guard'ом). */
export type AuthClient = ReturnType<typeof authorizedAuthClient>;

/** Результат guard-вызова: данные + действующий токен, либо passthrough-ошибка. */
export type GuardedResult<T> =
  { ok: true; data: T; token: string } | { ok: false; status: number; body: unknown };

/**
 * Есть ли роль admin в наборе ролей.
 *
 * @param roles - роли пользователя.
 * @returns true, если среди ролей есть admin.
 */
export function isAdminUser(roles: readonly string[] | undefined | null): boolean {
  return Array.isArray(roles) && roles.includes('admin');
}

/**
 * Обёртка вызова auth-service с прозрачным refresh: access-cookie → при 401
 * refresh и повтор. Возвращает данные и действующий токен (для последующих
 * вызовов users-service в том же запросе).
 *
 * @param event - событие h3.
 * @param op - вызов typed-клиента auth.
 * @returns данные с токеном или passthrough-ошибка сервиса.
 */
export async function callAuth<T>(
  event: H3Event,
  op: (client: AuthClient) => Promise<ApiCall<T>>,
): Promise<GuardedResult<T>> {
  const access = readAccess(event);

  if (access) {
    const res = await op(authorizedAuthClient(access, event));
    if (res.response?.status !== 401) {
      return toResult(res, access);
    }
  }

  const refreshed = await refreshSession(event);
  if (!refreshed.ok || !refreshed.tokens) {
    return {
      ok: false,
      status: 401,
      body: { code: 'unauthorized', message: 'authentication required' },
    };
  }

  const res = await op(authorizedAuthClient(refreshed.tokens.access_token, event));
  return toResult(res, refreshed.tokens.access_token);
}

/**
 *
 * @param res
 * @param token
 */
function toResult<T>(res: ApiCall<T>, token?: string): GuardedResult<T> {
  if (res.data !== undefined && res.response?.ok) {
    return { ok: true, data: res.data, token: token ?? '' };
  }
  return {
    ok: false,
    status: res.response?.status ?? 500,
    body: res.error ?? { code: 'internal', message: 'internal error' },
  };
}

/**
 * Aggregates auth users with users-service profiles (by id).
 *
 * @param users - страница пользователей из auth-service.
 * @param profiles - найденные профили (могут отсутствовать у части id).
 * @returns строки списка: user + профиль (или null).
 */
export function mergeProfiles(
  users: AdminUser[],
  profiles: UserProfile[] | null | undefined,
): AdminUserRow[] {
  const byId = new Map<string, UserProfile>();
  for (const profile of profiles ?? []) {
    byId.set(profile.user_id, profile);
  }
  return users.map((user) => ({ user, profile: byId.get(user.id) ?? null }));
}

/**
 * Guard админ-роутов: сессия + роль admin.
 *
 * @param event - событие h3.
 * @returns ok=false и passthrough-ошибку (401/403), если доступ запрещён.
 */
export async function requireAdmin(
  event: H3Event,
): Promise<
  { ok: true; token: string; user: AdminUser } | { ok: false; status: number; body: unknown }
> {
  const me = await callAuth(event, (client) => client.GET('/v1/me'));
  if (!me.ok) {
    return me;
  }
  if (!isAdminUser(me.data.roles)) {
    return {
      ok: false,
      status: 403,
      body: { code: 'forbidden', message: 'insufficient permissions' },
    };
  }
  return { ok: true, token: me.token, user: me.data };
}
