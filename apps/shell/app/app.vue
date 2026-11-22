<script setup lang="ts">
import AppHeader from './components/AppHeader.vue';

// Сессия и шапка — клиентские; SSR отдаёт каркас (ADR-003).
const year = new Date().getFullYear();
</script>

<template>
  <div class="shell">
    <AppHeader />

    <main class="shell-main">
      <NuxtPage />
    </main>

    <footer class="shell-footer">© {{ year }} Rosterra</footer>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.shell-main {
  flex: 1;
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
  padding: var(--ui-space-lg);
}

.shell-footer {
  padding: var(--ui-space-md) var(--ui-space-lg);
  text-align: center;
  color: var(--ui-color-text-muted);
  font-size: var(--ui-font-size-sm);
  border-top: 1px solid var(--ui-color-border);
}
</style>
