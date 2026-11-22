/**
 * MF-expose: ./UserEditPage — карточка пользователя (требует params.id).
 *
 * @param el - DOM-элемент host-страницы.
 * @param opts - navigate + params ({ id }).
 */
import * as mountLib from '../shared/lib/mount';
import UserEditScreen from '../screens/user-edit-screen.vue';

/**
 *
 * @param el
 * @param opts
 */
export function mount(el: HTMLElement, opts?: mountLib.MountOptions): void {
  mountLib.mountComponent(el, UserEditScreen, opts);
}
