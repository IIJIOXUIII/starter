/**
 * entities/session — клиент BFF-сессии (same-origin /api/auth/**).
 *
 * В федеративном режиме JS исполняется на origin shell'а — запросы same-origin,
 * cookies httpOnly работают нативно. Токены браузеру не отдаются.
 */

/** Пользователь в ответах BFF (форма совпадает с контрактом auth-service). */
export interface SessionUser {
  id: string;
  email: string;
  name: string | null;
  status: 'active' | 'blocked';
  roles: string[];
  email_verified: boolean;
  created_at: string;
  updated_at: string;
}

/** Error-envelope сервисов платформы. */
export interface ApiError {
  code: string;
  message: string;
  details?: { field: string; message: string }[];
}

/** Ошибка API с envelope — экраны ловят и рендерят message/details. */
export class SessionApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly details?: { field: string; message: string }[];

  constructor(status: number, envelope: ApiError) {
    super(envelope.message);
    this.name = 'SessionApiError';
    this.code = envelope.code;
    this.status = status;
    this.details = envelope.details;
  }
}

/**
 *
 * @param path
 * @param init
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: 'same-origin',
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
  });
  const body = (await response.json().catch(() => null)) as T | ApiError | null;

  if (!response.ok) {
    const envelope = (body ?? {
      code: 'internal',
      message: 'Сервис авторизации недоступен',
    }) as ApiError;
    throw new SessionApiError(response.status, envelope);
  }
  return body as T;
}

/** API сессии: все вызовы идут на BFF shell'а. */
export const sessionApi = {
  /**
   * Регистрация + авто-логин (cookies ставит BFF).
   *
   * @param email
   * @param password
   * @param name
   */
  register(email: string, password: string, name?: string): Promise<SessionUser> {
    return request('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name: name ?? undefined }),
    });
  },

  /**
   * Вход по паролю.
   *
   * @param email
   * @param password
   * @param rememberMe
   */
  login(email: string, password: string, rememberMe: boolean): Promise<SessionUser> {
    return request('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password, remember_me: rememberMe }),
    });
  },

  /** Текущий пользователь (401 → SessionApiError unauthorized). */
  me(): Promise<SessionUser> {
    return request('/api/auth/me');
  },

  /** Выход (идемпотентен). */
  logout(): Promise<void> {
    return request('/api/auth/logout', { method: 'POST' });
  },
};
