/**
 * Преобразует произвольный текст в URL-безопасный slug.
 *
 * Сохраняет буквы и цифры (включая кириллицу и другие Unicode-буквы),
 * прочие символы превращает в дефисы, схлопывает подряд идущие дефисы
 * и обрезает их по краям.
 *
 * @param input - Произвольная строка.
 * @returns Slug без ведущих/замыкающих дефисов; для строки без букв и цифр — пустая строка.
 *
 * @example
 * slugify('Hello World!');
 * // => 'hello-world'
 * @example
 * slugify('Привет, мир!');
 * // => 'привет-мир'
 */
export function slugify(input: string): string {
  return input
    .normalize('NFKD')
    .toLowerCase()
    .trim()
    .replace(/[^\p{L}\p{N}]+/gu, '-')
    .replace(/^-+|-+$/g, '');
}

/**
 * Обрезает строку до `maxLength` символов, добавляя суффикс (по умолчанию `…`).
 *
 * Итоговая длина строки (вместе с суффиксом) не превышает `maxLength`.
 *
 * @param input - Исходная строка.
 * @param maxLength - Максимальная длина результата, включая суффикс. Должна быть >= 1.
 * @param suffix - Суффикс, добавляемый к обрезанной строке.
 * @returns Сама строка, если она короче `maxLength`, иначе обрезанный вариант с суффиксом.
 * @throws {RangeError} Если `maxLength` меньше 1.
 *
 * @example
 * truncate('Длинный заголовок новости', 10);
 * // => 'Длинный н…'
 */
export function truncate(input: string, maxLength: number, suffix = '…'): string {
  if (maxLength < 1) {
    throw new RangeError(`maxLength must be >= 1, got ${maxLength}`);
  }
  if (input.length <= maxLength) {
    return input;
  }
  if (suffix.length >= maxLength) {
    return suffix.slice(0, maxLength);
  }
  return input.slice(0, maxLength - suffix.length) + suffix;
}
