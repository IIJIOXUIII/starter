<script setup lang="ts">
import { computed, inject } from 'vue';
import { RuiAlert, RuiButton, RuiCard, RuiTextField } from '@rosterra/ui';

import { useMe } from '../entities/session/use-me';
import { useEditUser } from '../features/edit-user/use-edit-user';
import { NAVIGATE_KEY, type MountOptions } from '../shared/lib/mount';

const navigate = inject<(path: string) => void>(NAVIGATE_KEY, (path) => {
  window.location.href = path;
});

const mountParams = inject<MountOptions['params']>('mfe:params', {});
const me = useMe();
const edit = useEditUser({ id: mountParams?.id ?? '' });

const canSave = computed(() => !edit.loading.value && !edit.notFound.value && me.isAdmin.value);

const roleOptions: Array<{ value: 'user' | 'admin'; label: string }> = [
  { value: 'user', label: 'user — обычный доступ' },
  { value: 'admin', label: 'admin — администратор' },
];

/**
 *
 * @param value
 * @param checked
 */
function onRoleChange(value: 'user' | 'admin', checked: boolean): void {
  const roles = new Set(edit.form.value.roles);
  if (checked) {
    roles.add(value);
  } else {
    roles.delete(value);
  }
  if (roles.size === 0) {
    roles.add('user'); // контракт: минимум одна роль
  }
  edit.form.value.roles = [...roles];
}
</script>

<template>
  <div class="edit-screen">
    <RuiAlert v-if="me.loading.value || edit.loading.value" variant="info"
      >Загрузка карточки…</RuiAlert
    >
    <RuiAlert v-else-if="!me.isAdmin.value" variant="error" role="alert">
      Недостаточно прав: раздел доступен только администраторам.
    </RuiAlert>
    <RuiAlert v-else-if="edit.notFound.value" variant="error" role="alert">
      Пользователь не найден.
    </RuiAlert>

    <RuiCard v-else :title="`Карточка: ${edit.email.value}`">
      <p class="edit-link">
        <a href="/admin/users" @click.prevent="navigate('/admin/users')"
          >← К списку пользователей</a
        >
      </p>

      <form class="edit-form" novalidate @submit.prevent="edit.save">
        <RuiTextField
          v-model="edit.form.value.displayName"
          label="Имя (профиль)"
          placeholder="Иван"
        />
        <RuiTextField
          v-model="edit.form.value.avatarUrl"
          label="Avatar URL"
          placeholder="https://…"
        />

        <label class="edit-field">
          Заметки (только admin)
          <textarea
            v-model="edit.form.value.notes"
            rows="3"
            placeholder="Служебные заметки"
          ></textarea>
        </label>

        <fieldset class="edit-field">
          <legend>Статус</legend>
          <label class="edit-radio"
            ><input v-model="edit.form.value.status" type="radio" value="active" /> active</label
          >
          <label class="edit-radio"
            ><input v-model="edit.form.value.status" type="radio" value="blocked" /> blocked
            (отзовёт сессии)</label
          >
        </fieldset>

        <fieldset class="edit-field">
          <legend>Роли</legend>
          <label v-for="role in roleOptions" :key="role.value" class="edit-radio">
            <input
              type="checkbox"
              :value="role.value"
              :checked="edit.form.value.roles.includes(role.value)"
              @change="onRoleChange(role.value, ($event.target as HTMLInputElement).checked)"
            />
            {{ role.label }}
          </label>
        </fieldset>

        <RuiAlert v-if="edit.savingError.value" variant="error" role="alert">{{
          edit.savingError.value
        }}</RuiAlert>
        <RuiAlert v-if="edit.saved.value" variant="success">Сохранено.</RuiAlert>

        <RuiButton type="submit" :disabled="!canSave || edit.saving.value">
          {{
            edit.saving.value
              ? 'Сохраняем…'
              : edit.dirty.value
                ? 'Сохранить изменения'
                : 'Сохранить'
          }}
        </RuiButton>
      </form>
    </RuiCard>
  </div>
</template>

<style scoped>
.edit-screen {
  width: min(640px, 100%);
}

.edit-link {
  margin: 0 0 var(--ui-space-md);
  font-size: var(--ui-font-size-sm);
}

.edit-link a {
  color: var(--ui-color-primary);
  text-decoration: none;
}

.edit-form {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.edit-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 0;
  border: none;
  font-size: var(--ui-font-size-sm);
  font-weight: 600;
}

.edit-field legend {
  padding: 0;
  margin-bottom: 6px;
}

.edit-field textarea {
  padding: var(--ui-space-sm) var(--ui-space-md);
  border: 1px solid var(--ui-color-border);
  border-radius: var(--ui-radius-md);
  font-family: var(--ui-font-family);
  font-size: var(--ui-font-size-md);
  resize: vertical;
}

.edit-radio {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 400;
  user-select: none;
}
</style>
