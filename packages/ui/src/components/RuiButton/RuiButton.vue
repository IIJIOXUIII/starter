<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  /** Визуальный вариант кнопки. */
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  /** Размер кнопки. */
  size?: 'sm' | 'md' | 'lg';
  /** HTML-тип кнопки. */
  type?: 'button' | 'submit' | 'reset';
  /** Блокирует взаимодействие. */
  disabled?: boolean;
  /** Растягивает кнопку на всю ширину контейнера. */
  block?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
  disabled: false,
  block: false,
});

const emit = defineEmits<{
  /** Пользователь кликнул по кнопке. */
  click: [event: MouseEvent];
}>();

const classes = computed(() => [
  'rui-button',
  `rui-button--${props.variant}`,
  `rui-button--${props.size}`,
  { 'rui-button--block': props.block },
]);
</script>

<template>
  <button
    :class="classes"
    :type="type"
    :disabled="disabled || undefined"
    @click="emit('click', $event)"
  >
    <slot />
  </button>
</template>

<style scoped>
.rui-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: var(--ui-radius-md);
  font-family: var(--ui-font-family);
  font-weight: 600;
  line-height: 1;
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    border-color 0.15s ease,
    color 0.15s ease,
    box-shadow 0.15s ease;
}

.rui-button:focus-visible {
  outline: none;
  box-shadow: var(--ui-focus-ring);
}

.rui-button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.rui-button--sm {
  height: 32px;
  padding: 0 var(--ui-space-md);
  font-size: var(--ui-font-size-sm);
}

.rui-button--md {
  height: 40px;
  padding: 0 var(--ui-space-lg);
  font-size: var(--ui-font-size-md);
}

.rui-button--lg {
  height: 48px;
  padding: 0 20px;
  font-size: var(--ui-font-size-lg);
}

.rui-button--primary {
  background-color: var(--ui-color-primary);
  color: #fff;
}

.rui-button--primary:hover:not(:disabled) {
  background-color: var(--ui-color-primary-hover);
}

.rui-button--secondary {
  background-color: var(--ui-color-bg);
  border-color: var(--ui-color-border);
  color: var(--ui-color-text);
}

.rui-button--secondary:hover:not(:disabled) {
  background-color: var(--ui-color-bg-subtle);
}

.rui-button--ghost {
  background-color: transparent;
  color: var(--ui-color-primary);
}

.rui-button--ghost:hover:not(:disabled) {
  background-color: var(--ui-color-primary-soft);
}

.rui-button--danger {
  background-color: var(--ui-color-danger);
  color: #fff;
}

.rui-button--danger:hover:not(:disabled) {
  background-color: var(--ui-color-danger-hover);
}

.rui-button--block {
  display: flex;
  width: 100%;
}
</style>
