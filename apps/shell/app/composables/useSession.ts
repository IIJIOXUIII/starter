/**
 * Состояние сессии shell'а: заполняется клиентом из BFF (/api/auth/me).
 * Сессия как таковая живёт в httpOnly cookies — тут только отображение.
 */
import type { SessionUser } from './session-user';

export const useSession = () => {
  const user = useState<SessionUser | null>('session:user', () => null);
  const loading = useState<boolean>('session:loading', () => true);

  /** Запрашивает /api/auth/me (клиентский вызов). */
  async function fetchMe(): Promise<void> {
    try {
      const result = await $fetch<{ user: SessionUser }>('/api/auth/me');
      user.value = result.user;
    } catch {
      user.value = null;
    } finally {
      loading.value = false;
    }
  }

  /** Выход: BFF отзывает сессию и чистит cookies. */
  async function logout(): Promise<void> {
    await $fetch('/api/auth/logout', { method: 'POST' }).catch(() => null);
    user.value = null;
  }

  return { user, loading, fetchMe, logout };
};
