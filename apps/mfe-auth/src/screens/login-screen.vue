<script setup lang="ts">
import { inject } from 'vue';
import { RuiAlert, RuiButton, RuiCard, RuiTextField } from '@rosterra/ui';

import { useLoginForm } from '../features/auth-by-password/use-login-form';
import { NAVIGATE_KEY } from '../shared/lib/mount';

const navigate = inject<(path: string) => void>(NAVIGATE_KEY, (path) => {
  window.location.href = path;
});

const { email, password, rememberMe, errors, serverError, isSubmitting, submit } = useLoginForm({
  onSuccess: () => navigate('/'),
});
</script>

<template>
  <RuiCard title="Вход" class="login-screen">
    <form class="login-screen__form" novalidate @submit.prevent="submit">
      <RuiAlert v-if="serverError" variant="error" role="alert">
        {{ serverError }}
      </RuiAlert>

      <RuiTextField
        v-model="email"
        label="Email"
        type="email"
        placeholder="user@example.com"
        autocomplete="email"
        :error="errors.email"
      />
      <RuiTextField
        v-model="password"
        label="Пароль"
        type="password"
        autocomplete="current-password"
        :error="errors.password"
      />

      <label class="login-screen__remember">
        <input v-model="rememberMe" type="checkbox" name="remember_me" />
        Запомнить меня
      </label>

      <RuiButton type="submit" block :disabled="isSubmitting">
        {{ isSubmitting ? 'Входим…' : 'Войти' }}
      </RuiButton>

      <p class="login-screen__alt">
        Нет аккаунта?
        <a href="/auth/register" @click.prevent="navigate('/auth/register')">Создать</a>
      </p>
    </form>
  </RuiCard>
</template>

<style scoped>
.login-screen {
  width: min(360px, 100%);
}

.login-screen__form {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.login-screen__remember {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--ui-font-size-sm);
  color: var(--ui-color-text);
  user-select: none;
}

.login-screen__alt {
  margin: 0;
  text-align: center;
  font-size: var(--ui-font-size-sm);
  color: var(--ui-color-text-muted);
}

.login-screen__alt a {
  color: var(--ui-color-primary);
}
</style>
