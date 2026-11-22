/**
 * MF-expose: ./UsersPage — остров админки (плейсхолдер до M5).
 *
 * @param el - DOM-элемент host-страницы.
 */
import { mountComponent } from '../shared/lib/mount';
import UsersPlaceholderScreen from '../screens/users-placeholder-screen.vue';

/**
 *
 * @param el
 */
export function mount(el: HTMLElement): void {
  mountComponent(el, UsersPlaceholderScreen);
}
