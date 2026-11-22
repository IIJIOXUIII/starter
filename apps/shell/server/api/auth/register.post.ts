/**
 * POST /api/auth/register — регистрация + авто-логин.
 *
 * Проксирует в auth-service, токены прячет в httpOnly cookies,
 * браузеру возвращает только { user }.
 */
import { z } from 'zod';

import { authClient } from '../../utils/api';
import { isSameOrigin, setSession } from '../../utils/session';
import { checkRate, clientIp } from '../../utils/ratelimit';
import { respondError, type ErrorEnvelope } from '../../utils/proxy';

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(8).max(128),
  name: z.string().max(100).optional(),
});

export default defineEventHandler(async (event) => {
  if (!isSameOrigin(event)) {
    return respondError(event, 403, {
      code: 'forbidden',
      message: 'cross-origin request blocked',
    } satisfies ErrorEnvelope);
  }
  const rate = checkRate(`${clientIp(event)}:register`, 10);
  if (!rate.ok) {
    setHeader(event, 'Retry-After', rate.retryAfterSeconds);
    return respondError(event, 429, {
      code: 'rate_limited',
      message: 'too many requests',
    } satisfies ErrorEnvelope);
  }

  const parsed = schema.safeParse(await readBody(event));
  if (!parsed.success) {
    return respondError(event, 400, {
      code: 'validation_failed',
      message: 'request validation failed',
      details: parsed.error.issues.map((i) => ({
        field: i.path.join('.'),
        message: i.message,
      })),
    } satisfies ErrorEnvelope);
  }

  const { data, error, response } = await authClient.POST('/v1/register', { body: parsed.data });
  if (error || !data) {
    return respondError(
      event,
      response.status,
      error ?? { code: 'internal', message: 'internal error' },
    );
  }

  setSession(event, data);
  setResponseStatus(event, 201);
  return { user: data.user };
});
