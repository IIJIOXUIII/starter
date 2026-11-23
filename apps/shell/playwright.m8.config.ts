import { defineConfig } from '@playwright/test';

// M8: полный критический путь одним прогоном.
export default defineConfig({
  testDir: 'tests/m8',
  use: {
    baseURL: 'http://localhost:3000',
  },
  workers: 1,
  retries: 1,
});
