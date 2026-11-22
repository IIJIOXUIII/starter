<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  /** Статус, определяющий цвет бейджа. */
  status: 'active' | 'blocked' | 'pending';
  /** Переопределяет текст (по умолчанию — человеческое имя статуса). */
  label?: string;
}

const props = withDefaults(defineProps<Props>(), { label: '' });

const defaultLabels: Record<Props['status'], string> = {
  active: 'Активен',
  blocked: 'Заблокирован',
  pending: 'Ожидает',
};

const text = computed(() => (props.label !== '' ? props.label : defaultLabels[props.status]));
</script>

<template>
  <span class="rui-status-badge" :class="`rui-status-badge--${status}`">{{ text }}</span>
</template>

<style scoped>
.rui-status-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: var(--ui-font-size-sm);
  font-weight: 600;
  line-height: 1.2;
  white-space: nowrap;
}

.rui-status-badge--active {
  background-color: #dcfce7;
  color: #166534;
}

.rui-status-badge--blocked {
  background-color: var(--ui-color-danger-soft);
  color: #991b1b;
}

.rui-status-badge--pending {
  background-color: #fef9c3;
  color: #854d0e;
}
</style>
