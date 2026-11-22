/**
 * features/edit-user — загрузка и сохранение карточки пользователя.
 */
import { computed, ref, watch, type Ref } from 'vue';

import {
  AdminApiError,
  adminApi,
  type AdminUser,
  type Profile,
} from '../../entities/admin-user/api';

export interface EditUserForm {
  displayName: string;
  avatarUrl: string;
  notes: string;
  status: 'active' | 'blocked';
  roles: ('user' | 'admin')[];
}

export interface EditUser {
  form: Ref<EditUserForm>;
  loading: Ref<boolean>;
  saving: Ref<boolean>;
  savingError: Ref<string>;
  saved: Ref<boolean>;
  notFound: Ref<boolean>;
  email: Ref<string>;
  /** Есть несохранённые изменения. */
  dirty: Ref<boolean>;
  save: () => Promise<void>;
}

export interface UseEditUserOptions {
  /** id пользователя из пути. */
  id: string;
}

/**
 * Форма из данных карточки.
 *
 * @param user
 * @param profile
 */
function toForm(user: AdminUser | null, profile: Profile | null): EditUserForm {
  return {
    displayName: profile?.display_name ?? user?.name ?? '',
    avatarUrl: profile?.avatar_url ?? '',
    notes: profile?.notes ?? '',
    status: user?.status ?? 'active',
    roles: user ? [...user.roles] : ['user'],
  };
}

/**
 *
 * @param options
 */
export function useEditUser(options: UseEditUserOptions): EditUser {
  const form = ref<EditUserForm>({
    displayName: '',
    avatarUrl: '',
    notes: '',
    status: 'active',
    roles: ['user'],
  });
  const loading = ref(true);
  const saving = ref(false);
  const savingError = ref('');
  const saved = ref(false);
  const notFound = ref(false);
  const email = ref('');

  let current: { user: AdminUser; profile: Profile | null } | null = null;

  watch(
    () => options.id,
    async (id) => {
      loading.value = true;
      notFound.value = false;
      saved.value = false;
      savingError.value = '';
      try {
        current = await adminApi.get(id);
        form.value = toForm(current.user, current.profile);
        email.value = current.user.email;
      } catch (err) {
        notFound.value = err instanceof AdminApiError;
        savingError.value =
          err instanceof AdminApiError ? err.message : 'Не удалось загрузить карточку';
      } finally {
        loading.value = false;
      }
    },
    { immediate: true },
  );

  const dirty = computed(
    () =>
      current !== null &&
      JSON.stringify(form.value) !== JSON.stringify(toForm(current.user, current.profile)),
  );

  /**
   *
   */
  async function save(): Promise<void> {
    if (current === null || saving.value) {
      return;
    }
    saving.value = true;
    savingError.value = '';
    saved.value = false;

    const before = toForm(current.user, current.profile);
    const body: Record<string, unknown> = {};

    if (form.value.status !== before.status) {
      body.status = form.value.status;
      if (form.value.status === 'blocked') {
        body.reason = 'blocked from admin panel';
      }
    }
    if (JSON.stringify([...form.value.roles].sort()) !== JSON.stringify([...before.roles].sort())) {
      body.roles = form.value.roles.length > 0 ? form.value.roles : ['user'];
    }
    if (
      form.value.displayName !== before.displayName ||
      form.value.avatarUrl !== before.avatarUrl ||
      form.value.notes !== before.notes
    ) {
      body.profile = {
        display_name: form.value.displayName,
        // '' = очистить поле (семантика контракта users-service).
        avatar_url: form.value.avatarUrl,
        notes: form.value.notes,
      };
    }

    try {
      const result = await adminApi.patch(options.id, body);
      // PATCH возвращает только изменённые части (user=null, если менялся лишь профиль).
      current = {
        user: result.user ?? current.user,
        profile: result.profile ?? current.profile,
      };
      form.value = toForm(current.user, current.profile);
      saved.value = true;
    } catch (err) {
      savingError.value =
        err instanceof AdminApiError ? err.message : 'Не удалось сохранить карточку';
    } finally {
      saving.value = false;
    }
  }

  return { form, loading, saving, savingError, saved, notFound, email, dirty, save };
}
