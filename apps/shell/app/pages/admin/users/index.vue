<script setup lang="ts">
useHead({ title: 'Админка — пользователи — Rosterra' });
</script>

<template>
  <ClientOnly>
    <RemotePage module-name="mfe_admin/UsersPage" />
    <template #fallback>
      <p class="page-fallback">Загрузка админки…</p>
    </template>
  </ClientOnly>
</template>

<style scoped>
.page-fallback {
  text-align: center;
  color: var(--ui-color-text-muted);
}
</style>
