/**
 * Module Federation 2.0 runtime для shell-хоста.
 *
 * Remotes грузятся по mf-manifest.json (runtime-загрузка, независимый деплой).
 * Инициализация — ленивая, только на клиенте (ADR-003: remotes рендерятся CSR).
 */
import { init, loadRemote } from '@module-federation/runtime';

/** Контракт MF-острова: mount создаёт независимое Vue-приложение. */
export interface RemotePage {
  mount(el: HTMLElement, opts?: { navigate?: (path: string) => void }): { unmount(): void };
}

let initialized = false;

/**
 *
 */
function ensureInit(): void {
  if (initialized) {
    return;
  }
  const config = useRuntimeConfig();
  const remotes: Array<{ name: string; entry: string }> = [
    { name: 'mfe_auth', entry: config.public.mfeAuthUrl },
    { name: 'mfe_admin', entry: config.public.mfeAdminUrl },
  ];
  init({ name: 'shell', remotes });
  initialized = true;
}

/**
 * Загружает страницу микрофронтенда.
 *
 * @param moduleName - 'mfe_auth/LoginPage' | 'mfe_admin/UsersPage' | …
 * @returns остров с методом mount().
 */
export async function loadRemotePage(moduleName: string): Promise<RemotePage> {
  ensureInit();
  const page = (await loadRemote(moduleName)) as RemotePage;
  if (typeof page?.mount !== 'function') {
    throw new Error(`remote ${moduleName} does not implement mount() contract`);
  }
  return page;
}
