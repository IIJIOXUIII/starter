import { vue } from '@rosterra/eslint-config';

/**
 * Корневая конфигурация ESLint — используется lint-staged (pre-commit), когда
 * ESLint вызывается из корня монорепо. Пакеты и приложения используют свои
 * конфиги (turbo run lint запускает eslint внутри каждой директории).
 */
export default vue;
