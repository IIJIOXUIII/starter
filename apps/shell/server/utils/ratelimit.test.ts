import { describe, expect, it } from 'vitest';
import { checkRate } from './ratelimit';

describe('checkRate', () => {
  it('пропускает первые N запросов и ограничивает дальше', () => {
    const now = 1_000_000;
    const key = 'ip:login';

    for (let i = 0; i < 10; i++) {
      const result = checkRate(key, 10, 60_000, now);
      expect(result.ok).toBe(true);
    }

    const blocked = checkRate(key, 10, 60_000, now);
    expect(blocked.ok).toBe(false);
    expect(blocked.retryAfterSeconds).toBeGreaterThan(0);
  });

  it('разные ключи считаются независимо', () => {
    const now = 2_000_000;
    expect(checkRate('a:login', 1, 60_000, now).ok).toBe(true);
    expect(checkRate('a:login', 1, 60_000, now).ok).toBe(false);
    expect(checkRate('b:login', 1, 60_000, now).ok).toBe(true);
  });

  it('после истечения окна счётчик сбрасывается', () => {
    const now = 3_000_000;
    expect(checkRate('c:login', 1, 60_000, now).ok).toBe(true);
    expect(checkRate('c:login', 1, 60_000, now).ok).toBe(false);
    // Окно 60s прошло.
    expect(checkRate('c:login', 1, 60_000, now + 61_000).ok).toBe(true);
  });
});
