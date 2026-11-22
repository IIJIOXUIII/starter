import { defineConfig } from '@playwright/test';

// Скриншотные тесты компонентов: каждый story рендерится в iframe.html
// собранного Storybook и сравнивается со снапшотом.
export default defineConfig({
  testDir: 'tests/visual',
  use: {
    baseURL: 'http://127.0.0.1:6007',
  },
  expect: {
    toHaveScreenshot: {
      maxDiffPixelRatio: 0.02,
    },
  },
  // Параллельность не нужна: страниц немного, стабильность важнее скорости.
  workers: 2,
});
