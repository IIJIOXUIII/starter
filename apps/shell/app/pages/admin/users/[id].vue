<script setup lang="ts">
const route = useRoute();
const userId = computed(() => String(route.params.id ?? ''));

useHead({ title: 'Админка — карточка пользователя — Rosterra' });
</script>

<template>
  <ClientOnly>
    <RemotePage module-name="mfe_admin/UserEditPage" :params="{ id: userId }" />
    <template #fallback>
      <p class="page-fallback">Загрузка карточки…</p>
    </template>
  </ClientOnly>
</template>

<style scoped>
.page-fallback {
  text-align: center;
  color: var(--ui-color-text-muted);
}
</style>
