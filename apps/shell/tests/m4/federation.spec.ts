/**
 * M4 smoke: федерация end-to-end.
 *
 * Требует запущенных: auth-service :8081, shell :3000, remotes :5173/:5174.
 * Проверяет, что shell реально грузит MF-острова и формы интерактивны.
 */
import { expect, test } from '@playwright/test';

test('страница входа загружает MF-остров и форма интерактивна', async ({ page }) => {
  await page.goto('/auth/login');

  // Остров смонтирован: поле email из дизайн-системы доступно по label.
  const email = page.getByLabel('Email');
  await expect(email).toBeVisible({ timeout: 20_000 });

  // Неверный пароль → ошибка от auth-service через BFF.
  await email.fill('m3test@rosterra.dev');
  await page.getByLabel('Пароль').fill('wrong-password');
  await page.getByRole('button', { name: 'Войти' }).click();
  await expect(page.getByRole('alert')).toContainText('Неверный email или пароль', {
    timeout: 10_000,
  });
});

test('страница регистрации загружает MF-остров', async ({ page }) => {
  await page.goto('/auth/register');
  await expect(page.getByRole('heading', { name: 'Регистрация' })).toBeVisible({ timeout: 20_000 });
  await expect(page.getByLabel('Имя')).toBeVisible();
});

test('успешный логин обновляет шапку shell', async ({ page }) => {
  const email = `m4-${Date.now()}@rosterra.dev`;
  const password = 'Strong-Pass-1';

  // Регистрируемся через остров.
  await page.goto('/auth/register');
  await page.getByLabel('Имя').fill('M4');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Пароль', { exact: true }).fill(password);
  await page.getByLabel('Повторите пароль').fill(password);
  await page.getByRole('button', { name: 'Создать аккаунт' }).click();

  // onSuccess → полный переход на /; SSR shell'а отдаёт каркас, шапка обновится клиентом.
  await page.waitForURL('**/');
  await expect(page.getByText(email)).toBeVisible({ timeout: 15_000 });

  // Админка для обычного пользователя — отказ RBAC-guard острова (M5: живая админка).
  await page.goto('/admin/users');
  await expect(page.getByText(/Недостаточно прав/)).toBeVisible({ timeout: 20_000 });
});

test('admin-остров грузится и защищён guard-ом (аноним)', async ({ page }) => {
  await page.goto('/admin/users');
  await expect(page.getByText(/Недостаточно прав/)).toBeVisible({ timeout: 20_000 });
});
