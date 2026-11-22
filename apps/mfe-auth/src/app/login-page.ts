/**
 * MF-expose: ./LoginPage — остров входа.
 *
 * Контракт: mount(el, opts) создаёт независимое Vue-приложение.
 *
 * @param el - DOM-элемент host-страницы.
 * @param opts - navigate для переходов средствами shell-роутера.
 */
import type { MountOptions } from '../shared/lib/mount';
import { mountComponent } from '../shared/lib/mount';
import LoginScreen from '../screens/login-screen.vue';

/**
 *
 * @param el
 * @param opts
 */
export function mount(el: HTMLElement, opts?: MountOptions): void {
  mountComponent(el, LoginScreen, opts);
}
