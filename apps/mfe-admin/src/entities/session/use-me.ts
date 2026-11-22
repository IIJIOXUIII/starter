/**
 * entities/session — текущий пользователь (для RBAC-guard острова).
 */
import { ref, type Ref } from 'vue';

/** Пользователь в ответах BFF. */
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

export interface MeState {
  user: Ref<SessionUser | null>;
  loading: Ref<boolean>;
  isAdmin: Ref<boolean>;
}

/**
 * Загружает /api/auth/me и вычисляет isAdmin.
 *
 * @returns реактивное состояние сессии.
 */
export function useMe(): MeState {
  const user = ref<SessionUser | null>(null);
  const loading = ref(true);
  const isAdmin = ref(false);

  void (async () => {
    try {
      const response = await fetch('/api/auth/me', { credentials: 'same-origin' });
      if (response.ok) {
        const body = (await response.json()) as { user: SessionUser };
        user.value = body.user;
        isAdmin.value = body.user.roles.includes('admin');
      }
    } catch {
      // неавторизован / BFF недоступен — гвард покажет ошибку доступа
    } finally {
      loading.value = false;
    }
  })();

  return { user, loading, isAdmin };
}
