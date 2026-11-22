/**
 * M5 smoke: живая админка (mfe-admin остров + BFF агрегация + users-service).
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

test('admin видит таблицу пользователей и открывает карточку', async ({ page }) => {
  await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);

  await page.goto('/admin/users');
  await expect(page.getByRole('heading', { name: 'Пользователи' })).toBeVisible({
    timeout: 20_000,
  });

  // Список пришёл из auth-service, профили — из users-service.
  // Проверяем ячейку таблицы (email администратора есть и в шапке — getByText дал бы strict violation).
  await expect(page.getByRole('cell', { name: ADMIN_EMAIL })).toBeVisible({ timeout: 15_000 });

  // Открываем карточку кликом по ячейке email в таблице (не в шапке).
  await page.getByRole('cell', { name: ADMIN_EMAIL }).click();
  await expect(page.getByRole('heading', { name: /Карточка/ })).toBeVisible({ timeout: 20_000 });
});

test('admin редактирует профиль пользователя (имя сохраняется в users-service)', async ({
  page,
}) => {
  await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);

  // Регистрируем свежего пользователя с уникальным email через UI регистрации.
  const email = `m5edit-${Date.now()}@rosterra.dev`;
  await page.goto('/auth/register');
  await page.getByLabel('Имя').fill('M5 Target');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Пароль', { exact: true }).fill('Strong-Pass-1');
  await page.getByLabel('Повторите пароль').fill('Strong-Pass-1');
  await page.getByRole('button', { name: 'Создать аккаунт' }).click();
  await page.waitForURL('**/');

  await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  await page.goto('/admin/users');

  // Ищем пользователя в списке (постранично — их немного, хватит первой страницы).
  await page.getByLabel('Поиск по email').fill(email);
  await expect(page.getByText(email)).toBeVisible({ timeout: 15_000 });

  // Открываем карточку кликом по ячейке email в таблице.
  await page.getByRole('cell', { name: email }).click();
  await expect(page.getByRole('heading', { name: /Карточка/ })).toBeVisible({ timeout: 20_000 });
  const nameField = page.getByLabel('Имя (профиль)');
  const newName = `Renamed-${Date.now()}`;
  await nameField.fill(newName);
  await page.getByRole('button', { name: /Сохранить/ }).click();
  await expect(page.getByText('Сохранено.')).toBeVisible({ timeout: 10_000 });

  // Возвращаемся к списку — новое имя в таблице.
  await page.getByText('К списку пользователей').click();
  await expect(page.getByRole('heading', { name: 'Пользователи' })).toBeVisible({
    timeout: 15_000,
  });
  await page.getByLabel('Поиск по email').fill(email);
  await expect(page.getByText(newName)).toBeVisible({ timeout: 15_000 });
});

test('не-admin получает отказ доступа (RBAC-guard острова)', async ({ page }) => {
  const email = `m5user-${Date.now()}@rosterra.dev`;
  await page.goto('/auth/register');
  await page.getByLabel('Имя').fill('M5 Plain');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Пароль', { exact: true }).fill('Strong-Pass-1');
  await page.getByLabel('Повторите пароль').fill('Strong-Pass-1');
  await page.getByRole('button', { name: 'Создать аккаунт' }).click();
  await page.waitForURL('**/');

  await page.goto('/admin/users');
  await expect(page.getByText(/Недостаточно прав/)).toBeVisible({ timeout: 20_000 });
});
