import { defineConfig } from '@playwright/test';

// M4 smoke: shell + MF-острова + BFF (нужны запущенные: auth :8081, shell :3000,
// remotes :5173/:5174). Отдельный конфиг, чтобы не смешивать со storybook-тестами.
export default defineConfig({
  testDir: 'tests/m4',
  use: {
    baseURL: 'http://localhost:3000',
  },
  workers: 1,
});
