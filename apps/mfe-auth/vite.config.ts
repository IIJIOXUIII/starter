import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vite';
import { federation } from '@module-federation/vite';

// Module Federation 2.0: remote с self-contained island'ами.
// Каждый expose экспортирует mount(el, opts) — монтирует независимое Vue-приложение.
// Shared-синглтоны не используем (ADR-003: session через cookies, состояние — внутри MFE).
export default defineConfig({
  plugins: [
    vue(),
    federation({
      name: 'mfe_auth',
      filename: 'remoteEntry.js',
      exposes: {
        './LoginPage': './src/app/login-page.ts',
        './RegisterPage': './src/app/register-page.ts',
      },
      manifest: true,
      remotes: {},
      shared: {},
    }),
  ],
  server: {
    cors: true, // shell (localhost:3000) грузит remoteEntry кросс-доменно
  },
  build: {
    target: 'esnext',
  },
});
