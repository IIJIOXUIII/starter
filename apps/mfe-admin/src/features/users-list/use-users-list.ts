/**
 * features/users-list — состояние списка пользователей (серверная пагинация).
 */
import { computed, onMounted, ref, type Ref } from 'vue';

import {
  AdminApiError,
  adminApi,
  type AdminUserRow,
  type ListParams,
} from '../../entities/admin-user/api';

export interface UsersList {
  rows: Ref<AdminUserRow[]>;
  total: Ref<number>;
  page: Ref<number>;
  perPage: Ref<number>;
  pageCount: Ref<number>;
  query: Ref<string>;
  status: Ref<'all' | 'active' | 'blocked'>;
  loading: Ref<boolean>;
  error: Ref<string>;
  reload: () => Promise<void>;
  nextPage: () => Promise<void>;
  prevPage: () => Promise<void>;
}

export interface UseUsersListOptions {
  perPage?: number;
}

/**
 *
 * @param options
 */
export function useUsersList(options: UseUsersListOptions = {}): UsersList {
  const perPage = options.perPage ?? 20;
  const rows = ref<AdminUserRow[]>([]);
  const total = ref(0);
  const page = ref(1);
  const query = ref('');
  const status = ref<'all' | 'active' | 'blocked'>('all');
  const loading = ref(false);
  const error = ref('');

  /**
   *
   */
  async function reload(): Promise<void> {
    loading.value = true;
    error.value = '';
    try {
      const params: ListParams = { page: page.value, per_page: perPage };
      if (query.value.trim() !== '') {
        params.query = query.value.trim();
      }
      if (status.value !== 'all') {
        params.status = status.value;
      }
      const data = await adminApi.list(params);
      rows.value = data.items;
      total.value = data.total;
    } catch (err) {
      error.value = err instanceof AdminApiError ? err.message : 'Не удалось загрузить список';
      rows.value = [];
      total.value = 0;
    } finally {
      loading.value = false;
    }
  }

  const pageCount = computed(() => Math.max(1, Math.ceil(total.value / perPage)));

  /**
   *
   */
  async function nextPage(): Promise<void> {
    if (page.value < pageCount.value) {
      page.value += 1;
      await reload();
    }
  }

  /**
   *
   */
  async function prevPage(): Promise<void> {
    if (page.value > 1) {
      page.value -= 1;
      await reload();
    }
  }

  onMounted(() => {
    void reload();
  });

  return {
    rows,
    total,
    page,
    perPage: ref(perPage),
    pageCount,
    query,
    status,
    loading,
    error,
    reload,
    nextPage,
    prevPage,
  };
}
