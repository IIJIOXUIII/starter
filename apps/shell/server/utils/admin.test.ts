import { describe, expect, it } from 'vitest';

import { isAdminUser, mergeProfiles, type AdminUser, type UserProfile } from './admin';

const user = (id: string, email: string): AdminUser =>
  ({
    id,
    email,
    name: null,
    status: 'active',
    roles: ['user'],
    email_verified: false,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }) as AdminUser;

const profile = (userId: string, displayName: string): UserProfile =>
  ({
    user_id: userId,
    display_name: displayName,
    avatar_url: null,
    notes: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }) as UserProfile;

describe('isAdminUser', () => {
  it('находит роль admin', () => {
    expect(isAdminUser(['user', 'admin'])).toBe(true);
  });

  it('без admin — false', () => {
    expect(isAdminUser(['user'])).toBe(false);
    expect(isAdminUser([])).toBe(false);
    expect(isAdminUser(undefined)).toBe(false);
    expect(isAdminUser(null)).toBe(false);
  });
});

describe('mergeProfiles', () => {
  it('соединяет пользователей с профилями по id', () => {
    const rows = mergeProfiles(
      [user('u1', 'a@b.dev'), user('u2', 'c@d.dev')],
      [profile('u2', 'Cathy'), profile('u1', 'Anna')],
    );
    expect(rows).toHaveLength(2);
    expect(rows[0].user.id).toBe('u1');
    expect(rows[0].profile?.display_name).toBe('Anna');
    expect(rows[1].profile?.display_name).toBe('Cathy');
  });

  it('пользователь без профиля получает null (событие в полёте)', () => {
    const rows = mergeProfiles([user('u1', 'a@b.dev')], []);
    expect(rows[0].profile).toBeNull();
  });

  it('не падает на undefined/null профилях', () => {
    expect(mergeProfiles([user('u1', 'a@b.dev')], undefined)).toHaveLength(1);
    expect(mergeProfiles([], null)).toEqual([]);
  });

  it('сохраняет порядок списка auth', () => {
    const rows = mergeProfiles(
      [user('u2', 'c@d.dev'), user('u1', 'a@b.dev')],
      [profile('u1', 'A')],
    );
    expect(rows.map((r) => r.user.id)).toEqual(['u2', 'u1']);
    expect(rows[1].profile?.display_name).toBe('A');
  });
});
