<script setup lang="ts">
interface Props {
  /** Заголовок карточки (необязательный). */
  title?: string;
}

withDefaults(defineProps<Props>(), { title: '' });
</script>

<template>
  <section class="rui-card">
    <h2 v-if="title !== ''" class="rui-card__title">{{ title }}</h2>
    <slot />
  </section>
</template>

<style scoped>
.rui-card {
  padding: var(--ui-space-lg);
  border: 1px solid var(--ui-color-border);
  border-radius: var(--ui-radius-lg);
  background-color: var(--ui-color-bg);
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.04);
}

.rui-card__title {
  margin: 0 0 var(--ui-space-md);
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--ui-color-text);
}
</style>
