<script setup lang="ts">
useHead({ title: 'Вход — Rosterra' });
</script>

<template>
  <ClientOnly>
    <RemotePage module-name="mfe_auth/LoginPage" />
    <template #fallback>
      <p class="page-fallback">Загрузка формы входа…</p>
    </template>
  </ClientOnly>
</template>

<style scoped>
.page-fallback {
  text-align: center;
  color: var(--ui-color-text-muted);
}
</style>
