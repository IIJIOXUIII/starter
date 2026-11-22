/**
 * entities/admin-user — клиент admin-API BFF (same-origin /api/admin/**).
 *
 * В федеративном режиме JS исполняется на origin shell'а — запросы same-origin,
 * cookie-сессия работает нативно. Токены браузеру не отдаются.
 */

/** Пользователь auth-service (в ответах BFF). */
export interface AdminUser {
  id: string;
  email: string;
  name: string | null;
  status: 'active' | 'blocked';
  roles: ('user' | 'admin')[];
  email_verified: boolean;
  created_at: string;
  updated_at: string;
}

/** Профиль users-service. */
export interface Profile {
  user_id: string;
  display_name: string;
  avatar_url: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

/** Строка агрегированного списка. */
export interface AdminUserRow {
  user: AdminUser;
  profile: Profile | null;
}

/** Страница списка. */
export interface UsersPage {
  items: AdminUserRow[];
  total: number;
  page: number;
  per_page: number;
}

/** Error-envelope сервисов платформы. */
export interface ApiError {
  code: string;
  message: string;
  details?: { field: string; message: string }[];
}

/** Ошибка API с envelope. */
export class AdminApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(status: number, envelope: ApiError) {
    super(envelope.message);
    this.name = 'AdminApiError';
    this.code = envelope.code;
    this.status = status;
  }
}

/** Параметры списка. */
export interface ListParams {
  query?: string;
  status?: 'active' | 'blocked';
  page?: number;
  per_page?: number;
}

/** Тело PATCH карточки. */
export interface PatchBody {
  status?: 'active' | 'blocked';
  reason?: string;
  roles?: ('user' | 'admin')[];
  profile?: {
    display_name?: string;
    avatar_url?: string;
    notes?: string;
  };
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
    const envelope = (body ?? { code: 'internal', message: 'internal error' }) as ApiError;
    throw new AdminApiError(response.status, envelope);
  }
  return body as T;
}

/** Admin API поверх BFF. */
export const adminApi = {
  /**
   * Агрегированный список с серверной пагинацией.
   *
   * @param params
   */
  list(params: ListParams): Promise<UsersPage> {
    const qs = new URLSearchParams();
    if (params.query) {
      qs.set('query', params.query);
    }
    if (params.status) {
      qs.set('status', params.status);
    }
    qs.set('page', String(params.page ?? 1));
    qs.set('per_page', String(params.per_page ?? 20));
    return request(`/api/admin/users?${qs.toString()}`);
  },

  /**
   * Карточка пользователя + профиль.
   *
   * @param id
   */
  get(id: string): Promise<{ user: AdminUser; profile: Profile | null }> {
    return request(`/api/admin/users/${id}`);
  },

  /**
   * Сохранение карточки (статус/роли/профиль).
   *
   * @param id
   * @param body
   */
  patch(id: string, body: PatchBody): Promise<{ user: AdminUser; profile: Profile | null }> {
    return request(`/api/admin/users/${id}`, { method: 'PATCH', body: JSON.stringify(body) });
  },
};
