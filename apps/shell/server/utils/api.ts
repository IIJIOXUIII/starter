/**
 * Типобезопасные клиенты Go-сервисов для BFF.
 *
 * Базовые URL берутся из окружения (в compose/K8s — внутренний DNS),
 * локальные дефолты соответствуют docker-compose.
 */
import { createAuthClient, createUsersClient } from '@rosterra/api-client';

const AUTH_BASE = process.env.NUXT_AUTH_BASE ?? 'http://localhost:8081';
const USERS_BASE = process.env.NUXT_USERS_BASE ?? 'http://localhost:8082';

/** Клиент auth-service (регистрация, сессии, роли). */
export const authClient = createAuthClient(AUTH_BASE);

/**
 * Клиент auth-service с Authorization-заголовком (openapi-fetch ставит
 * заголовки через middleware, а не через опции запроса).
 *
 * @param token - access-JWT из cookie сессии.
 * @returns клиент, добавляющий Bearer ко всем вызовам.
 */
export function authorizedAuthClient(token: string) {
  const client = createAuthClient(AUTH_BASE);
  client.use({
    onRequest({ request }) {
      request.headers.set('Authorization', `Bearer ${token}`);
      return request;
    },
  });
  return client;
}

/** Клиент users-service (профили) — пригодится в M5. */
export const usersClient = createUsersClient(USERS_BASE);
