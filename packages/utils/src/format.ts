/**
 * Форматирует размер в байтах в человекочитаемую строку (единицы СИ: KB, MB, GB…).
 *
 * @param bytes - Размер в байтах. Отрицательные значения запрещены.
 * @param decimals - Количество знаков после запятой (0–3).
 * @returns Отформатированная строка вида `"1.5 KB"`.
 * @throws {RangeError} Если `bytes` отрицательный или `decimals` вне диапазона 0–3.
 *
 * @example
 * formatBytes(0);
 * // => '0 B'
 * @example
 * formatBytes(1536);
 * // => '1.5 KB'
 */
export function formatBytes(bytes: number, decimals = 1): string {
  if (bytes < 0) {
    throw new RangeError(`bytes must be non-negative, got ${bytes}`);
  }
  if (!Number.isInteger(decimals) || decimals < 0 || decimals > 3) {
    throw new RangeError(`decimals must be an integer in [0..3], got ${decimals}`);
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'] as const;
  if (bytes === 0) {
    return '0 B';
  }

  const exponent = Math.min(Math.floor(Math.log2(bytes) / 10), units.length - 1);
  const value = bytes / 1024 ** exponent;
  const formatted = value.toFixed(exponent === 0 ? 0 : decimals);

  return `${formatted} ${units[exponent]}`;
}
