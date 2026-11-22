import { defineConfig } from '@playwright/test';

// M5 smoke: живая админка (нужны запущенные: auth :8081, users :8082, shell :3000,
// remotes :5173/:5174).
export default defineConfig({
  testDir: 'tests/m5',
  use: {
    baseURL: 'http://localhost:3000',
  },
  workers: 1,
});
