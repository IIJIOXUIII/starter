import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vite';
import { federation } from '@module-federation/vite';

// Module Federation 2.0: remote админки (остров с self-contained Vue).
export default defineConfig({
  plugins: [
    vue(),
    federation({
      name: 'mfe_admin',
      filename: 'remoteEntry.js',
      exposes: {
        './UsersPage': './src/app/users-page.ts',
        './UserEditPage': './src/app/user-edit-page.ts',
      },
      manifest: true,
      remotes: {},
      shared: {},
    }),
  ],
  server: {
    cors: true,
  },
  build: {
    target: 'esnext',
  },
});
