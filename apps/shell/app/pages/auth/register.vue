<script setup lang="ts">
useHead({ title: 'Регистрация — Rosterra' });
</script>

<template>
  <ClientOnly>
    <RemotePage module-name="mfe_auth/RegisterPage" />
    <template #fallback>
      <p class="page-fallback">Загрузка формы регистрации…</p>
    </template>
  </ClientOnly>
</template>

<style scoped>
.page-fallback {
  text-align: center;
  color: var(--ui-color-text-muted);
}
</style>
