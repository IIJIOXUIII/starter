/**
 * Standalone-режим mfe-admin.
 */
import '@rosterra/ui/tokens.css';

import * as UsersPage from './app/users-page';

const root = document.createElement('div');
root.className = 'standalone-root';
document.body.appendChild(root);
UsersPage.mount(root);
