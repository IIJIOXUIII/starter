/**
 * Скриншотные (visual regression) тесты дизайн-системы.
 *
 * Запуск: pnpm test:storybook:visual
 * (собирает Storybook, поднимает статику на :6007 и прогоняет этот spec)
 *
 * Базовые снапшоты лежат в tests/visual/__screenshots__/ — они зависят от ОС
 * (рендер шрифтов), поэтому генерируются на каждой платформе отдельно:
 *   pnpm exec playwright test --update-snapshots
 */
import { expect, test } from '@playwright/test';
import { readFileSync } from 'node:fs';

interface StoryIndexEntry {
  id: string;
  type: string;
  title: string;
  name: string;
}

interface StoryIndex {
  entries: Record<string, StoryIndexEntry>;
}

const indexPath = new URL('../../storybook-static/index.json', import.meta.url);
const index: StoryIndex = JSON.parse(readFileSync(indexPath, 'utf8'));

const stories = Object.entries(index.entries)
  // Документация (--docs) — не визуальный контент.
  .filter(([id, entry]) => entry.type === 'story' && !id.includes('--docs'))
  .map(([id, entry]) => ({ id, name: `${entry.title} · ${entry.name}` }));

test.describe('Компоненты дизайн-системы — визуальная регрессия', () => {
  for (const story of stories) {
    test(story.name, async ({ page }) => {
      await page.goto(`/iframe.html?id=${story.id}&viewMode=story`);
      // Дождаться гидрации и отрисовки (шрифты/анимации).
      await page.waitForTimeout(200);
      await expect(page.locator('#storybook-root')).toHaveScreenshot({
        fullPage: true,
        animations: 'disabled',
      });
    });
  }
});
