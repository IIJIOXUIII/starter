/**
 * MF-expose: ./UsersPage — список пользователей (таблица, RBAC-guard внутри).
 *
 * @param el - DOM-элемент host-страницы.
 * @param opts - navigate для переходов средствами shell-роутера.
 */
import * as mountLib from '../shared/lib/mount';
import UsersTableScreen from '../screens/users-table-screen.vue';

/**
 *
 * @param el
 * @param opts
 */
export function mount(el: HTMLElement, opts?: mountLib.MountOptions): void {
  mountLib.mountComponent(el, UsersTableScreen, opts);
}
