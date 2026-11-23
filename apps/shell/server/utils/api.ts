/**
 * Типобезопасные клиенты Go-сервисов для BFF.
 *
 * Базовые URL берутся из окружения (в compose/K8s — внутренний DNS),
 * локальные дефолты соответствуют docker-compose.
 *
 * Наблюдаемость: W3C traceparent из запроса браузера пробрасывается в
 * сервисы — трейсы BFF → auth/users коррелируются (M6).
 */
import type { H3Event } from 'h3';
import { createAuthClient, createUsersClient } from '@rosterra/api-client';

const AUTH_BASE = process.env.NUXT_AUTH_BASE ?? 'http://localhost:8081';
const USERS_BASE = process.env.NUXT_USERS_BASE ?? 'http://localhost:8082';

/** Заголовки трассировки, пробрасываемые в Go-сервисы. */
const TRACE_HEADERS = ['traceparent', 'tracestate', 'baggage'] as const;

/**
 * Извлекает W3C-заголовки трассировки из входящего запроса.
 *
 * @param event - событие h3.
 * @returns объект заголовков (пустой, если браузер не начал трейс).
 */
export function traceHeaders(event: H3Event): Record<string, string> {
  const result: Record<string, string> = {};
  for (const name of TRACE_HEADERS) {
    const value = getRequestHeader(event, name);
    if (value) {
      result[name] = value;
    }
  }
  return result;
}

/** Клиент auth-service (регистрация, сессии, роли). */
export const authClient = createAuthClient(AUTH_BASE);

/**
 * Клиент auth-service с Authorization-заголовком (openapi-fetch ставит
 * заголовки через middleware, а не через опции запроса).
 *
 * @param token - access-JWT из cookie сессии.
 * @param event - событие h3 (опционально: проброс traceparent в сервис).
 * @returns клиент, добавляющий Bearer и trace-заголовки ко всем вызовам.
 */
export function authorizedAuthClient(token: string, event?: H3Event) {
  const client = createAuthClient(AUTH_BASE);
  client.use({
    onRequest({ request }) {
      request.headers.set('Authorization', `Bearer ${token}`);
      if (event) {
        for (const [name, value] of Object.entries(traceHeaders(event))) {
          request.headers.set(name, value);
        }
      }
      return request;
    },
  });
  return client;
}

/** Клиент users-service (профили). */
export const usersClient = createUsersClient(USERS_BASE);

/**
 * Клиент users-service с Authorization-заголовком.
 *
 * @param token - access-JWT из cookie сессии.
 * @param event - событие h3 (опционально: проброс traceparent в сервис).
 * @returns клиент, добавляющий Bearer и trace-заголовки ко всем вызовам.
 */
export function authorizedUsersClient(token: string, event?: H3Event) {
  const client = createUsersClient(USERS_BASE);
  client.use({
    onRequest({ request }) {
      request.headers.set('Authorization', `Bearer ${token}`);
      if (event) {
        for (const [name, value] of Object.entries(traceHeaders(event))) {
          request.headers.set(name, value);
        }
      }
      return request;
    },
  });
  return client;
}
