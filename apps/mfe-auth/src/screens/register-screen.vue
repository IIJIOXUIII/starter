<script setup lang="ts">
import { inject } from 'vue';
import { RuiAlert, RuiButton, RuiCard, RuiTextField } from '@rosterra/ui';

import { useRegisterForm } from '../features/register-by-password/use-register-form';
import { NAVIGATE_KEY } from '../shared/lib/mount';

const navigate = inject<(path: string) => void>(NAVIGATE_KEY, (path) => {
  window.location.href = path;
});

const { name, email, password, confirm, errors, serverError, isSubmitting, submit } =
  useRegisterForm({
    onSuccess: () => navigate('/'),
  });
</script>

<template>
  <RuiCard title="Регистрация" class="register-screen">
    <form class="register-screen__form" novalidate @submit.prevent="submit">
      <RuiAlert v-if="serverError" variant="error" role="alert">
        {{ serverError }}
      </RuiAlert>

      <RuiTextField
        v-model="name"
        label="Имя"
        placeholder="Иван"
        autocomplete="name"
        :error="errors.name"
      />
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
        autocomplete="new-password"
        :error="errors.password"
      />
      <RuiTextField
        v-model="confirm"
        label="Повторите пароль"
        type="password"
        autocomplete="new-password"
        :error="errors.confirm"
      />

      <RuiButton type="submit" block :disabled="isSubmitting">
        {{ isSubmitting ? 'Создаём аккаунт…' : 'Создать аккаунт' }}
      </RuiButton>

      <p class="register-screen__alt">
        Уже есть аккаунт?
        <a href="/auth/login" @click.prevent="navigate('/auth/login')">Войти</a>
      </p>
    </form>
  </RuiCard>
</template>

<style scoped>
.register-screen {
  width: min(400px, 100%);
}

.register-screen__form {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.register-screen__alt {
  margin: 0;
  text-align: center;
  font-size: var(--ui-font-size-sm);
  color: var(--ui-color-text-muted);
}

.register-screen__alt a {
  color: var(--ui-color-primary);
}
</style>
