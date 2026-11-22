// Shell — SSR-хост микрофронтов и BFF (server/api).
// Микрофронты подключаются через Module Federation в M4 (см. docs/ARCHITECTURE.md, §7).
export default defineNuxtConfig({
  compatibilityDate: '2026-11-01',

  devtools: { enabled: true },

  css: ['@rosterra/ui/tokens.css'],

  typescript: {
    strict: true,
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
