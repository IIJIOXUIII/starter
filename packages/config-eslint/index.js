import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import pluginVue from 'eslint-plugin-vue';
import jsdoc from 'eslint-plugin-jsdoc';
import prettier from 'eslint-config-prettier';

/** Общие игноры всех пакетов монорепо. */
const ignores = {
  ignores: [
    '**/node_modules/**',
    '**/dist/**',
    '**/.nuxt/**',
    '**/.output/**',
    '**/.turbo/**',
    '**/coverage/**',
    '**/gen/**',
    '**/storybook-static/**',
  ],
};

/** База для чистого TypeScript (утилиты, пакеты без Vue). */
export const base = tseslint.config(
  ignores,
  js.configs.recommended,
  ...tseslint.configs.recommended,
  jsdoc.configs['flat/recommended-typescript'],
  prettier,
  {
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrors: 'none' },
      ],
      '@typescript-eslint/consistent-type-imports': ['error', { prefer: 'type-imports' }],
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      // Стиль JSDoc: одна пустая строка после описания, без ограничений между тегами.
      'jsdoc/tag-lines': ['error', 'any', { startLines: 1 }],
    },
  },
);

/** Конфигурация для Vue-приложений и библиотек компонентов. */
export const vue = tseslint.config(
  ...base,
  ...pluginVue.configs['flat/recommended'],
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        // vue-eslint-parser парсит шаблон, TS-парсер — содержимое <script>.
        parser: tseslint.parser,
        extraFileExtensions: ['.vue'],
        sourceType: 'module',
      },
    },
    rules: {
      'vue/multi-word-component-names': 'off',
      // eslint-recommended отключает no-undef только для *.ts — для .vue делаем это сами:
      // наличие глобалов проверяет TypeScript, а не ESLint.
      'no-undef': 'off',
    },
  },
  // Prettier — строго последним: отключает стилистические vue-правила, конфликтующие с форматтером.
  prettier,
);
