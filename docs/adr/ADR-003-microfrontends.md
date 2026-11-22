# ADR-003: Микрофронты — Module Federation 2.0, remotes на CSR

- Статус: принято (M0, реализация — M4)
- Дата: 2026-11-22

## Контекст

Нужны независимо разрабатываемые и деплоимые фронтенд-приложения (auth, admin,
будущие домены) с единым shell'ом, дизайн-системой и сессией.

## Решение

- **Shell** — Nuxt 4: layout, роутинг верхнего уровня, auth-guard, SSR.
- **Remotes** — Vue 3 + Vite, сборка через `@module-federation/vite` (MF 2.0):
  `mfe-auth`, `mfe-admin`; каждый remote деплоится независимо.
- **Модель «островов» (уточнение v1.1)**: remotes НЕ шарят vue/pinia синглтоны с shell —
  каждый expose экспортирует `mount(el, opts)`, создающее независимое Vue-приложение
  в переданном DOM-узле. Причина: надёжный shared scope между Nuxt-хостом и Vite-remote
  хрупок; сессия живёт в httpOnly cookies BFF, поэтому общий реактивный стейт
  shell ↔ remote не нужен (после auth-действий — полный переход, SSR shell читает cookies).
- Remotes поддерживают **standalone-режим** (свой dev-сервер и index.html).
- Remotes регистрируют маршруты в shell контрактом `installRoutes`-страниц:
  страницы shell (`/auth/login`, `/auth/register`, `/admin/users`) — тонкие обёртки
  над `<RemotePage module-name="mfe_auth/LoginPage">`.
- Содержимое remotes рендерится **на клиенте**; SSR остаётся у shell
  (каркас, мета, SSR-запросы через BFF).

## Альтернативы

- **single-spa** — зрелый, но более низкоуровневый DX.
- **Native import maps** — без тулинга MF, больше ручной работы.
- **Nuxt Layers** — compile-time композиция, не даёт независимого деплоя remotes.

## Последствия

- Независимый релиз каждого микрофронта.
- Компромисс: remotes без SSR (стандартно для MF); критичные данные тянутся
  SSR-слоем shell.
- Риск зрелости `@module-federation/vite` контролируем: подключаем в M4
  на минимальном кейсе; fallback-план — single-spa/import maps.
