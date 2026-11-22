/**
 * POST /api/auth/login — вход по email и паролю.
 *
 * remember_me=true → refresh cookie на 30 дней, иначе 12 часов.
 */
import { z } from 'zod';

import { authClient } from '../../utils/api';
import { isSameOrigin, setAuthCookies } from '../../utils/session';
import { checkRate, clientIp } from '../../utils/ratelimit';
import { respondError, type ErrorEnvelope } from '../../utils/proxy';

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(1).max(128),
  remember_me: z.boolean().optional(),
});

export default defineEventHandler(async (event) => {
  if (!isSameOrigin(event)) {
    return respondError(event, 403, {
      code: 'forbidden',
      message: 'cross-origin request blocked',
    } satisfies ErrorEnvelope);
  }
  const rate = checkRate(`${clientIp(event)}:login`, 10);
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

  const { data, error, response } = await authClient.POST('/v1/login', {
    body: {
      email: parsed.data.email,
      password: parsed.data.password,
      remember_me: parsed.data.remember_me ?? false,
    },
  });
  if (error || !data) {
    return respondError(
      event,
      response.status,
      error ?? { code: 'internal', message: 'internal error' },
    );
  }

  setAuthCookies(event, data);
  return { user: data.user };
});
