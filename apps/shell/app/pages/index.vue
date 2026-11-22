<script setup lang="ts">
// Автоимпорты Nuxt (useHead) + явный импорт компонента из дизайн-системы:
// библиотека @rosterra/ui — обычный пакет, её компоненты не автоподхватываются.
import { RuiButton } from '@rosterra/ui';

useHead({ title: 'Rosterra — главная' });
</script>

<template>
  <section>
    <h1>Добро пожаловать в Rosterra</h1>
    <p>
      Стартовый проект микросервисной платформы: Go-сервисы, BFF и микрофронты. Страницы входа и
      регистрации появятся вместе с микрофронтом auth (M4).
    </p>
    <RuiButton>Кнопка из дизайн-системы</RuiButton>
  </section>
</template>
