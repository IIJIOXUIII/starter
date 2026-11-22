/**
 * shared/lib — контракт монтирования микрофронт-островов.
 *
 * Каждая MF-страница экспортирует mount(el, opts): создаёт собственное
 * Vue-приложение (island) и монтирует в переданный элемент.
 */
import { createApp, type App } from 'vue';

/** Опции монтирования острова. */
export interface MountOptions {
  /**
   * Навигация средствами хоста (shell-роутер). В standalone — обычная навигация.
   */
  navigate?: (path: string) => void;
}

/** Ключ provide для экранов. */
export const NAVIGATE_KEY = 'mfe:navigate';

/**
 * Дефолтная навигация — полный переход (standalone-режим).
 *
 * @param path
 */
export const defaultNavigate = (path: string): void => {
  window.location.href = path;
};

/**
 * Монтирует компонент как независимое приложение с навигацией в provide.
 *
 * @param el - DOM-элемент host-страницы.
 * @param component - корневой компонент экрана.
 * @param opts - опции монтирования.
 * @returns инстанс приложения (для unmount host'ом).
 */
export function mountComponent(
  el: HTMLElement,
  component: Parameters<typeof createApp>[0],
  opts?: MountOptions,
): App {
  const app = createApp(component);
  app.provide(NAVIGATE_KEY, opts?.navigate ?? defaultNavigate);
  app.mount(el);
  return app;
}
