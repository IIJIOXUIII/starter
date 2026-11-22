import type { StorybookConfig } from '@storybook/vue3-vite';

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  addons: ['@storybook/addon-docs', '@storybook/addon-a11y'],
  framework: {
    name: '@storybook/vue3-vite',
    options: {},
  },
  // vue3-vite не подключает @vitejs/plugin-vue сам — добавляем явно.
  async viteFinal(config) {
    const vuePlugin = (await import('@vitejs/plugin-vue')).default;
    config.plugins = config.plugins ?? [];
    config.plugins.push(vuePlugin());
    return config;
  },
};

export default config;
