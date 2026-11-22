# ADR-003: Микрофронты — Module Federation 2.0, remotes на CSR

- Статус: принято (M0, реализация — M4)
- Дата: 2026-11-22

## Контекст

Нужны независимо разрабатываемые и деплоимые фронтенд-приложения (auth, admin,
будущие домены) с единым shell'ом, дизайн-системой и сессией.

## Решение

- **Shell** — Nuxt 4: layout, роутинг верхнего уровня, auth-guard, SSR.
- **Remotes** — Vue 3 + Vite, сборка через `@module-federation/vite` (MF 2.0):
  `mfe-auth`, `mfe-admin`; shared-синглтоны `vue`/`pinia`.
- Remotes поддерживают **standalone-режим** (свой dev-сервер и index.html).
- Remotes регистрируют маршруты в роутере shell контрактом `installRoutes(router)`.
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
