<script setup lang="ts">
import { FlexRender, createColumnHelper, getCoreRowModel, useVueTable } from '@tanstack/vue-table';
import { computed, h, inject } from 'vue';
import { RuiAlert, RuiButton, RuiCard, RuiStatusBadge, RuiTextField } from '@rosterra/ui';

import type { AdminUserRow } from '../entities/admin-user/api';
import { useMe } from '../entities/session/use-me';
import { useUsersList } from '../features/users-list/use-users-list';
import { NAVIGATE_KEY } from '../shared/lib/mount';

const navigate = inject<(path: string) => void>(NAVIGATE_KEY, (path) => {
  window.location.href = path;
});

const me = useMe();
const list = useUsersList();

/** Даты в читаемом формате (остров рендерится на клиенте). */
const fmt = new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' });

const columnHelper = createColumnHelper<AdminUserRow>();

const columns = computed(() => [
  columnHelper.accessor((row) => row.user.email, {
    id: 'email',
    header: 'Email',
  }),
  columnHelper.accessor((row) => row.profile?.display_name ?? row.user.name ?? '', {
    id: 'display_name',
    header: 'Имя',
  }),
  columnHelper.accessor((row) => row.user.status, {
    id: 'status',
    header: 'Статус',
    cell: (info) => h(RuiStatusBadge, { status: info.getValue() }),
  }),
  columnHelper.accessor((row) => row.user.roles, {
    id: 'roles',
    header: 'Роли',
    cell: (info) => info.getValue().join(', '),
  }),
  columnHelper.accessor((row) => row.user.created_at, {
    id: 'created_at',
    header: 'Создан',
    cell: (info) => fmt.format(new Date(info.getValue())),
  }),
]);

const table = useVueTable({
  get data() {
    return list.rows.value;
  },
  get columns() {
    return columns.value;
  },
  getCoreRowModel: getCoreRowModel(),
  manualPagination: true,
  get pageCount() {
    return list.pageCount.value;
  },
});

let searchTimer: ReturnType<typeof setTimeout> | undefined;

/** Поиск с debounce — сервер фильтрует по подстроке email. */
function onSearch(): void {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    list.page.value = 1;
    void list.reload();
  }, 350);
}

/** Фильтр по статусу. */
async function onStatusChange(): Promise<void> {
  list.page.value = 1;
  await list.reload();
}

/**
 * Открыть карточку пользователя средствами shell-навигации.
 *
 * @param id
 */
function openUser(id: string): void {
  navigate(`/admin/users/${id}`);
}
</script>

<template>
  <div class="users-screen">
    <RuiAlert v-if="me.loading.value" variant="info">Проверка прав…</RuiAlert>
    <RuiAlert v-else-if="!me.isAdmin.value" variant="error" role="alert">
      Недостаточно прав: раздел доступен только администраторам.
    </RuiAlert>

    <template v-else>
      <RuiCard title="Пользователи">
        <div class="users-toolbar">
          <RuiTextField
            v-model="list.query.value"
            label="Поиск по email"
            placeholder="user@example.com"
            @update:model-value="onSearch"
          />
          <label class="users-status">
            Статус
            <select v-model="list.status.value" @change="onStatusChange">
              <option value="all">все</option>
              <option value="active">активные</option>
              <option value="blocked">заблокированные</option>
            </select>
          </label>
          <RuiButton variant="secondary" :disabled="list.loading.value" @click="list.reload()"
            >Обновить</RuiButton
          >
        </div>

        <RuiAlert v-if="list.error.value" variant="error" role="alert">{{
          list.error.value
        }}</RuiAlert>

        <table class="users-table">
          <thead>
            <tr>
              <th v-for="header in table.getFlatHeaders()" :key="header.id">
                <FlexRender :render="header.column.columnDef.header" :props="header.getContext()" />
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in table.getRowModel().rows"
              :key="row.id"
              class="users-row"
              @click="openUser(row.original.user.id)"
            >
              <td v-for="cell in row.getVisibleCells()" :key="cell.id">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </td>
            </tr>
            <tr v-if="!list.loading.value && list.rows.value.length === 0">
              <td colspan="5" class="users-empty">Пользователи не найдены.</td>
            </tr>
          </tbody>
        </table>

        <div class="users-pagination">
          <span>Всего: {{ list.total.value }}</span>
          <span class="users-pagination-spacer"></span>
          <RuiButton
            size="sm"
            variant="secondary"
            :disabled="list.page.value <= 1"
            @click="list.prevPage()"
          >
            ← Назад
          </RuiButton>
          <span>стр. {{ list.page.value }} из {{ list.pageCount.value }}</span>
          <RuiButton
            size="sm"
            variant="secondary"
            :disabled="list.page.value >= list.pageCount.value"
            @click="list.nextPage()"
          >
            Вперёд →
          </RuiButton>
        </div>
      </RuiCard>
    </template>
  </div>
</template>

<style scoped>
.users-screen {
  width: 100%;
}

.users-toolbar {
  display: flex;
  gap: var(--ui-space-md);
  align-items: flex-end;
  flex-wrap: wrap;
  margin-bottom: var(--ui-space-md);
}

.users-toolbar > :first-child {
  flex: 1;
  min-width: 220px;
}

.users-status {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: var(--ui-font-size-sm);
  font-weight: 600;
}

.users-status select {
  height: 40px;
  padding: 0 var(--ui-space-md);
  border: 1px solid var(--ui-color-border);
  border-radius: var(--ui-radius-md);
  background: var(--ui-color-bg);
  font-family: var(--ui-font-family);
}

.users-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--ui-font-size-sm);
}

.users-table th {
  text-align: left;
  padding: var(--ui-space-sm) var(--ui-space-md);
  border-bottom: 2px solid var(--ui-color-border);
  color: var(--ui-color-text-muted);
}

.users-table td {
  padding: var(--ui-space-sm) var(--ui-space-md);
  border-bottom: 1px solid var(--ui-color-border);
}

.users-row {
  cursor: pointer;
}

.users-row:hover {
  background: var(--ui-color-primary-soft);
}

.users-empty {
  text-align: center;
  color: var(--ui-color-text-muted);
  padding: var(--ui-space-lg);
}

.users-pagination {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
  color: var(--ui-color-text-muted);
  font-size: var(--ui-font-size-sm);
}

.users-pagination-spacer {
  flex: 1;
}
</style>
