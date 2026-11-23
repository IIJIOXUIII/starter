/**
 * M8 — полный критический путь платформы ОДНИМ прогоном.
 *
 * Регистрация → авто-вход → выход → вход → RBAC-отказ → блокировка админом
 * → отказ входа → разблокировка + выдача роли admin → доступ к админке → выход.
 *
 * Требует запущенных: auth :8081, users :8082, shell :3000, remotes :5173/:5174.
 */
import { expect, test, type Page } from '@playwright/test';

const ADMIN_EMAIL = 'anna@rosterra.dev';
const ADMIN_PASSWORD = 'Strong-Pass-1';

/**
 *
 * @param page
 * @param email
 * @param password
 */
async function login(page: Page, email: string, password: string): Promise<void> {
  await page.goto('/auth/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Пароль').fill(password);
  await page.getByRole('button', { name: 'Войти' }).click();
  await page.waitForURL('**/');
  await expect(page.getByText(email)).toBeVisible({ timeout: 15_000 });
}

/**
 *
 * @param page
 */
async function logout(page: Page): Promise<void> {
  await page.getByRole('button', { name: 'Выйти' }).click();
  await expect(page.getByRole('link', { name: 'Вход' })).toBeVisible({ timeout: 15_000 });
}

/**
 *
 * @param page
 * @param email
 */
async function openUserCard(page: Page, email: string): Promise<void> {
  await page.goto('/admin/users');
  await expect(page.getByRole('heading', { name: 'Пользователи' })).toBeVisible({
    timeout: 20_000,
  });
  await page.getByLabel('Поиск по email').fill(email);
  await expect(page.getByRole('cell', { name: email })).toBeVisible({ timeout: 15_000 });
  await page.getByRole('cell', { name: email }).click();
  await expect(page.getByRole('heading', { name: /Карточка/ })).toBeVisible({ timeout: 20_000 });
}

test('критический путь: регистрация → сессии → RBAC → блокировка → роли → восстановление', async ({
  page,
}) => {
  const email = `m8-${Date.now()}@rosterra.dev`;
  const password = 'Strong-Pass-1';

  // 1. Регистрация через остров → авто-вход (cookies BFF), шапка обновляется.
  await page.goto('/auth/register');
  await page.getByLabel('Имя').fill('M8 Journey');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Пароль', { exact: true }).fill(password);
  await page.getByLabel('Повторите пароль').fill(password);
  await page.getByRole('button', { name: 'Создать аккаунт' }).click();
  await page.waitForURL('**/');
  await expect(page.getByText(email)).toBeVisible({ timeout: 15_000 });

  // 2. Выход → шапка показывает гостевые ссылки.
  await logout(page);

  // 3. Повторный вход — сессия восстанавливается.
  await login(page, email, password);

  // 4. Обычный пользователь в админку не пускают (RBAC-guard острова).
  await page.goto('/admin/users');
  await expect(page.getByText(/Недостаточно прав/)).toBeVisible({ timeout: 20_000 });

  // 5. Админ блокирует пользователя через карточку.
  await logout(page);
  await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  await openUserCard(page, email);
  await page.getByRole('radio', { name: /blocked/ }).check();
  await page.getByRole('button', { name: /Сохранить/ }).click();
  await expect(page.getByText('Сохранено.')).toBeVisible({ timeout: 10_000 });

  // 6. Заблокированный пользователь не может войти.
  await logout(page);
  await page.goto('/auth/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Пароль').fill(password);
  await page.getByRole('button', { name: 'Войти' }).click();
  await expect(page.getByRole('alert')).toContainText('заблокирован', { timeout: 10_000 });

  // 7. Админ разблокирует и выдаёт роль admin одним сохранением.
  await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  await openUserCard(page, email);
  await page.getByRole('radio', { name: /active/ }).check();
  await page.getByRole('checkbox', { name: /admin —/ }).check();
  await page.getByRole('button', { name: /Сохранить/ }).click();
  await expect(page.getByText('Сохранено.')).toBeVisible({ timeout: 10_000 });

  // 8. Пользователь с новой ролью видит таблицу админки (RBAC-positive).
  await logout(page);
  await login(page, email, password);
  await page.goto('/admin/users');
  await expect(page.getByRole('heading', { name: 'Пользователи' })).toBeVisible({
    timeout: 20_000,
  });
  await expect(page.getByRole('cell', { name: email })).toBeVisible({ timeout: 15_000 });

  // 9. Финальный выход — сессия чистится.
  await logout(page);
  await expect(page.getByRole('link', { name: 'Регистрация' })).toBeVisible({ timeout: 10_000 });
});
