// Конфиг test-runner: interaction-тесты (play-функции) + a11y-сканы (addon-a11y).
// Скриншотные (visual) тесты — отдельный playwright-spec: pnpm test:storybook:visual.
import type { TestRunnerConfig } from '@storybook/test-runner';

const config: TestRunnerConfig = {
  tagsToExclude: [], // все stories участвуют; визуальные снапшоты — отдельно
};

export default config;
