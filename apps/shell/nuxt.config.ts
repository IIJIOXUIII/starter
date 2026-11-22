// Shell — SSR-хост микрофронтов и BFF (server/api).
// Микрофронты подключаются через Module Federation в M4 (см. docs/ARCHITECTURE.md, §7).
export default defineNuxtConfig({
  compatibilityDate: '2026-11-01',

  devtools: { enabled: true },

  css: ['@rosterra/ui/tokens.css'],

  typescript: {
    strict: true,
  },

  // BFF: адреса Go-сервисов (переопределяются env: NUXT_AUTH_BASE, NUXT_USERS_BASE).
  runtimeConfig: {
    authBase: 'http://localhost:8081',
    usersBase: 'http://localhost:8082',
  },

  // TS-исходники workspace-пакетов должны транспилироваться и в SSR, и в клиент.
  build: {
    transpile: ['@rosterra/api-client', '@rosterra/ui', '@rosterra/utils'],
  },

  app: {
    head: {
      title: 'Rosterra',
      htmlAttrs: { lang: 'ru' },
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
    },
  },
});
