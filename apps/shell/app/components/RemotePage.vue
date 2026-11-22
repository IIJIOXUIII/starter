<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { RuiCard } from '@rosterra/ui';

import { loadRemotePage } from '../composables/mf';

/**
 * Тонкая обёртка страницы shell'а: грузит MF-остров в ClientOnly-режиме
 * и монтирует его в собственный DOM-узел.
 */
const props = defineProps<{
  /** Имя модуля remote'а, например 'mfe_auth/LoginPage'. */
  moduleName: string;
  /** Параметры пути для острова (например, { id } карточки админки). */
  params?: Record<string, string>;
}>();

const hostEl = ref<HTMLElement | null>(null);
const state = ref<'loading' | 'ready' | 'error'>('loading');
let unmount: (() => void) | null = null;

onMounted(async () => {
  try {
    const page = await loadRemotePage(props.moduleName);
    if (hostEl.value) {
      const app = page.mount(hostEl.value, {
        navigate: (path) => window.location.assign(path),
        params: props.params,
      });
      unmount = () => app.unmount();
      state.value = 'ready';
    }
  } catch (err) {
    console.error('failed to load remote', props.moduleName, err);
    state.value = 'error';
  }
});

onBeforeUnmount(() => unmount?.());
</script>

<template>
  <div ref="hostEl" class="remote-host"></div>
  <RuiCard v-if="state === 'loading'" title="Загрузка…">Модуль загружается.</RuiCard>
  <RuiCard v-else-if="state === 'error'" title="Модуль недоступен">
    Не удалось загрузить микрофронтенд. Убедитесь, что он запущен, и обновите страницу.
  </RuiCard>
</template>

<style scoped>
.remote-host {
  display: flex;
  justify-content: center;
  width: 100%;
}
</style>
