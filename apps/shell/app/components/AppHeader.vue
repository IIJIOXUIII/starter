<script setup lang="ts">
import { RuiButton, RuiStatusBadge } from '@rosterra/ui';

import { useSession } from '../composables/useSession';

const session = useSession();
onMounted(() => session.fetchMe());
</script>

<template>
  <header class="shell-header">
    <nav class="shell-nav">
      <NuxtLink to="/" class="shell-logo">Rosterra</NuxtLink>
      <NuxtLink to="/">Главная</NuxtLink>
      <NuxtLink v-if="session.user.value" to="/admin/users">Админка</NuxtLink>

      <span class="shell-nav-spacer"></span>

      <template v-if="session.loading.value">
        <span class="shell-muted">…</span>
      </template>
      <template v-else-if="session.user.value">
        <RuiStatusBadge
          v-if="session.user.value.roles.includes('admin')"
          status="active"
          label="admin"
        />
        <span class="shell-user">{{ session.user.value.email }}</span>
        <RuiButton size="sm" variant="secondary" @click="session.logout()">Выйти</RuiButton>
      </template>
      <template v-else>
        <NuxtLink to="/auth/login">Вход</NuxtLink>
        <NuxtLink to="/auth/register">Регистрация</NuxtLink>
      </template>
    </nav>
  </header>
</template>

<style scoped>
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

.shell-nav-spacer {
  flex: 1;
}

.shell-muted {
  color: var(--ui-color-text-muted);
}

.shell-user {
  font-size: var(--ui-font-size-sm);
  color: var(--ui-color-text-muted);
}
</style>
