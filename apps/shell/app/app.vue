<script setup lang="ts">
// Каркас shell: навигация и слот страниц.
// С M4 сюда подключаются микрофронты (auth/admin) через Module Federation.
const year = new Date().getFullYear();
</script>

<template>
  <div class="shell">
    <header class="shell-header">
      <nav class="shell-nav">
        <NuxtLink to="/" class="shell-logo">Rosterra</NuxtLink>
        <NuxtLink to="/">Главная</NuxtLink>
        <!-- Маршруты микрофронтов появятся в M4 -->
        <span class="shell-nav-note">auth / admin — подключаются в M4</span>
      </nav>
    </header>

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

.shell-header {
  border-bottom: 1px solid var(--ui-color-border);
  background: var(--ui-color-bg);
}

.shell-nav {
  display: flex;
  align-items: center;
  gap: var(--ui-space-lg);
  max-width: 1200px;
  margin: 0 auto;
  padding: var(--ui-space-md) var(--ui-space-lg);
}

.shell-logo {
  font-weight: 700;
  font-size: 1.1rem;
  color: var(--ui-color-primary);
  text-decoration: none;
}

.shell-nav a {
  color: var(--ui-color-text);
  text-decoration: none;
}

.shell-nav a:hover {
  color: var(--ui-color-primary);
}

.shell-nav-note {
  margin-left: auto;
  font-size: var(--ui-font-size-sm);
  color: var(--ui-color-text-muted);
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
