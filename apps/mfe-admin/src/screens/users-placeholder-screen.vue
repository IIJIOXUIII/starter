<script setup lang="ts">
import { RuiAlert, RuiCard } from '@rosterra/ui';

// Плейсхолдер админки; полноценные таблица и редактирование — в M5
// (users-service + BFF admin-роуты уже реализованы в auth-service).
</script>

<template>
  <RuiCard title="Админка — пользователи">
    <RuiAlert variant="info">
      Экран управления пользователями появится в M5: таблица с пагинацией, редактирование профиля,
      статусы и роли.
    </RuiAlert>
  </RuiCard>
</template>
