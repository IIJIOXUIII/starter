import { describe, expect, it } from 'vitest';
import { slugify, truncate } from './string';

describe('slugify', () => {
  it('переводит текст в нижний регистр и заменяет разделители на дефисы', () => {
    expect(slugify('Hello World!')).toBe('hello-world');
  });

  it('поддерживает кириллицу', () => {
    expect(slugify('Привет, мир!')).toBe('привет-мир');
  });

  it('схлопывает подряд идущие разделители', () => {
    expect(slugify('a   ---  b')).toBe('a-b');
  });

  it('обрезает дефисы по краям', () => {
    expect(slugify('--тест--')).toBe('тест');
  });

  it('возвращает пустую строку для текста без букв и цифр', () => {
    expect(slugify('!!! ???')).toBe('');
    expect(slugify('')).toBe('');
  });
});

describe('truncate', () => {
  it('не изменяет короткую строку', () => {
    expect(truncate('abc', 10)).toBe('abc');
  });

  it('обрезает длинную строку с учётом длины суффикса', () => {
    expect(truncate('abcdefghij', 5)).toBe('abcd…');
    expect(truncate('abcdefghij', 5, '...')).toBe('ab...');
  });

  it('возвращает суффикс, если maxLength меньше его длины', () => {
    expect(truncate('abcdef', 2, '...')).toBe('..');
  });

  it('бросает RangeError для maxLength < 1', () => {
    expect(() => truncate('abc', 0)).toThrow(RangeError);
  });
});
