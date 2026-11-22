<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  /** Вариант уведомления: ошибка / успех / информация. */
  variant?: 'error' | 'success' | 'info';
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'info',
});

/** ARIA-роль: ошибки — alert (прерывание скринридера), остальное — status. */
const role = computed(() => (props.variant === 'error' ? 'alert' : 'status'));

const icon = computed(() => {
  if (props.variant === 'error') return '⚠';
  if (props.variant === 'success') return '✓';
  return 'ℹ';
});
</script>

<template>
  <div class="rui-alert" :class="`rui-alert--${variant}`" :role="role">
    <span class="rui-alert__icon" aria-hidden="true">{{ icon }}</span>
    <div class="rui-alert__content"><slot /></div>
  </div>
</template>

<style scoped>
.rui-alert {
  display: flex;
  gap: var(--ui-space-md);
  align-items: flex-start;
  padding: var(--ui-space-md) var(--ui-space-lg);
  border-radius: var(--ui-radius-md);
  font-size: var(--ui-font-size-md);
  line-height: 1.4;
}

.rui-alert__icon {
  flex-shrink: 0;
  font-weight: 700;
}

.rui-alert--error {
  background-color: var(--ui-color-danger-soft);
  color: #991b1b;
}

.rui-alert--success {
  background-color: #dcfce7;
  color: #166534;
}

.rui-alert--info {
  background-color: var(--ui-color-primary-soft);
  color: var(--ui-color-primary-700);
}
</style>
