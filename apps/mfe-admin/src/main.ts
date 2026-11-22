/**
 * Standalone-режим mfe-admin: список (?), карточка (?screen=edit&id=...).
 */
import '@rosterra/ui/tokens.css';

import * as UsersPage from './app/users-page';
import * as UserEditPage from './app/user-edit-page';
import { defaultNavigate, type MountOptions } from './shared/lib/mount';

const params = new URLSearchParams(window.location.search);
const isEdit = params.get('screen') === 'edit';
const page = isEdit ? UserEditPage : UsersPage;

const root = document.createElement('div');
root.className = 'standalone-root';
document.body.appendChild(root);

// Дефолтный id для standalone-режима карточки.
const opts: MountOptions | undefined = isEdit
  ? { navigate: defaultNavigate, params: { id: params.get('id') ?? '' } }
  : undefined;
page.mount(root, opts);
