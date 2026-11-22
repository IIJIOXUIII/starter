import { describe, expect, it } from 'vitest';
import { formatBytes } from './format';

describe('formatBytes', () => {
  it('форматирует ноль как "0 B"', () => {
    expect(formatBytes(0)).toBe('0 B');
  });

  it('не добавляет дробную часть для байтов', () => {
    expect(formatBytes(1)).toBe('1 B');
    expect(formatBytes(1023)).toBe('1023 B');
  });

  it('переводит в KB и применяет округление', () => {
    expect(formatBytes(1024)).toBe('1.0 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(1536, 2)).toBe('1.50 KB');
  });

  it('поддерживает старшие единицы', () => {
    expect(formatBytes(5 * 1024 ** 2)).toBe('5.0 MB');
    expect(formatBytes(3 * 1024 ** 3)).toBe('3.0 GB');
  });

  it('бросает RangeError для отрицательных значений', () => {
    expect(() => formatBytes(-1)).toThrow(RangeError);
  });

  it('бросает RangeError для недопустимого decimals', () => {
    expect(() => formatBytes(1, 4)).toThrow(RangeError);
    expect(() => formatBytes(1, 1.5)).toThrow(RangeError);
  });
});
