/**
 * Простейший in-memory rate limiter (fixed window) для auth-эндпоинтов.
 *
 * Дев/старт-уровень: на проде добавится Redis-бэкенд (общий на инстансы BFF)
 * и/или rate limit на edge (Traefik). Интерфейс — pure-функция, тестируется юнит-тестами.
 */
import type { H3Event } from 'h3';

/** Результат проверки лимита. */
export interface RateResult {
  /** true — запрос пропускается. */
  ok: boolean;
  /** Секунды до сброса окна (для заголовка Retry-After). */
  retryAfterSeconds: number;
}

const buckets = new Map<string, { count: number; resetAt: number }>();

/**
 * Проверяет и увеличивает счётчик запросов по ключу.
 *
 * @param key - уникальный ключ (IP + маршрут).
 * @param limit - максимум запросов в окно.
 * @param windowMs - размер окна в миллисекундах.
 * @param now - текущее время (инжектится для тестов).
 * @returns результат проверки: пропускать запрос или нет.
 */
export function checkRate(
  key: string,
  limit = 10,
  windowMs = 60_000,
  now = Date.now(),
): RateResult {
  const bucket = buckets.get(key);

  if (!bucket || bucket.resetAt <= now) {
    buckets.set(key, { count: 1, resetAt: now + windowMs });
    return { ok: true, retryAfterSeconds: 0 };
  }

  bucket.count += 1;
  if (bucket.count > limit) {
    return { ok: false, retryAfterSeconds: Math.ceil((bucket.resetAt - now) / 1000) };
  }
  return { ok: true, retryAfterSeconds: 0 };
}

/**
 * IP запроса (учитывает X-Forwarded-For за прокси).
 *
 * @param event - событие h3.
 * @returns IP клиента или 'unknown'.
 */
export function clientIp(event: H3Event): string {
  const xff = getHeader(event, 'x-forwarded-for');
  if (xff) {
    return xff.split(',')[0]!.trim();
  }
  return event.node.req.socket.remoteAddress ?? 'unknown';
}
