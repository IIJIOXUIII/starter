/**
 * Standalone-режим mfe-auth: независимая разработка/отладка экранов
 * без shell'а. Экран переключается через ?screen=register.
 */
import '@rosterra/ui/tokens.css';

import * as LoginPage from './app/login-page';
import * as RegisterPage from './app/register-page';
import { defaultNavigate } from './shared/lib/mount';

const isRegister = new URLSearchParams(window.location.search).get('screen') === 'register';
const page = isRegister ? RegisterPage : LoginPage;
const root = document.createElement('div');
root.className = 'standalone-root';
document.body.appendChild(root);

// Мини-шапка standalone-режима.
const nav = document.createElement('div');
nav.className = 'standalone-nav';
nav.innerHTML = `
  <a href="/">Вход</a>
  <a href="/?screen=register">Регистрация</a>
`;
document.body.prepend(nav);

const style = document.createElement('style');
style.textContent = `
  .standalone-nav { display: flex; gap: 16px; padding: 16px; justify-content: center; }
  .standalone-nav a { color: var(--ui-color-primary); text-decoration: none; }
`;
document.head.appendChild(style);

page.mount(root, { navigate: defaultNavigate });
