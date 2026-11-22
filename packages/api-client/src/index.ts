/**
 * Типобезопасные HTTP-клиенты сервисов Rosterra для BFF (Nuxt server routes).
 *
 * Типы `paths`/`components` генерируются из контрактов OpenAPI
 * (`api/openapi/*.yaml`) командой `pnpm gen:api`. Ручные правки gen/ запрещены.
 */
import createClient, { type Middleware } from 'openapi-fetch';
import type { components as AuthComponents, paths as AuthPaths } from './gen/auth';
import type { components as UsersComponents, paths as UsersPaths } from './gen/users';

export type { Middleware };

/** Типы контракта auth-service (schema User, Error, SessionTokens и т.д.). */
export type AuthApi = AuthComponents['schemas'];
/** Типы контракта users-service (schema Profile, Error и т.д.). */
export type UsersApi = UsersComponents['schemas'];

/**
 * Создаёт типобезопасный клиент auth-service.
 *
 * @param baseUrl - Базовый URL сервиса (в dev: `http://localhost:8081`).
 * @returns Клиент openapi-fetch с типами из `api/openapi/auth.yaml`.
 *
 * @example
 * const auth = createAuthClient('http://localhost:8081');
 * const { data, error } = await auth.POST('/v1/login', { body: { email, password } });
 * // data: SessionTokens | undefined; error: Error | undefined
 */
export function createAuthClient(baseUrl: string) {
  return createClient<AuthPaths>({ baseUrl });
}

/**
 * Создаёт типобезопасный клиент users-service.
 *
 * @param baseUrl - Базовый URL сервиса (в dev: `http://localhost:8082`).
 * @returns Клиент openapi-fetch с типами из `api/openapi/users.yaml`.
 *
 * @example
 * const users = createUsersClient('http://localhost:8082');
 * const { data } = await users.GET('/v1/profiles/{user_id}', { params: { path: { user_id } } });
 */
export function createUsersClient(baseUrl: string) {
  return createClient<UsersPaths>({ baseUrl });
}
