/** Пользователь в ответах BFF (форма контракта auth-service). */
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
