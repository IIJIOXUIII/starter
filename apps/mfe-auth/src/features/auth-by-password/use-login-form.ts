/**
 * features/auth-by-password — вход по email и паролю.
 *
 * vee-validate + zod: валидация полей, состояние отправки,
 * человекочитаемое маппинг-ошибок сервера в serverError.
 */
import { ref, type Ref } from 'vue';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import * as z from 'zod';

import { SessionApiError, sessionApi, type SessionUser } from '../../entities/session/api';

/** zod-схема формы входа. */
const loginSchema = z.object({
  email: z
    .string()
    .min(1, 'Укажите email')
    .regex(/^[^\s@]+@[^\s@]+\.[^\s@]+$/, 'Некорректный email'),
  password: z.string().min(1, 'Укажите пароль'),
  remember_me: z.boolean(),
});

/**
 * Человекочитаемый текст доменных ошибок auth-service.
 *
 * @param err
 */
function mapAuthError(err: SessionApiError): string {
  switch (err.code) {
    case 'invalid_credentials':
      return 'Неверный email или пароль';
    case 'account_locked':
      return 'Аккаунт заблокирован. Обратитесь к администратору';
    case 'rate_limited':
      return 'Слишком много попыток входа — попробуйте позже';
    default:
      return err.message;
  }
}

export interface UseLoginFormOptions {
  /** Успешный вход (cookies уже установлены BFF). */
  onSuccess: (user: SessionUser) => void;
}

export interface LoginForm {
  email: Ref<string | undefined>;
  password: Ref<string | undefined>;
  rememberMe: Ref<boolean | undefined>;
  /** Ошибки полей (vee-validate). */
  errors: Ref<Partial<Record<'email' | 'password' | 'remember_me', string>>>;
  serverError: Ref<string>;
  isSubmitting: Ref<boolean>;
  submit: (event?: Event) => Promise<void>;
}

/**
 *
 * @param options
 */
export function useLoginForm(options: UseLoginFormOptions): LoginForm {
  const serverError = ref('');

  const form = useForm({
    validationSchema: toTypedSchema(loginSchema),
    initialValues: { email: '', password: '', remember_me: false },
  });

  const email = form.defineField('email')[0];
  const password = form.defineField('password')[0];
  const rememberMe = form.defineField('remember_me')[0];

  const submitHandler = form.handleSubmit(async (values) => {
    serverError.value = '';
    try {
      const user = await sessionApi.login(values.email, values.password, values.remember_me);
      options.onSuccess(user);
    } catch (err) {
      serverError.value =
        err instanceof SessionApiError
          ? mapAuthError(err)
          : 'Сервис авторизации недоступен, попробуйте позже';
    }
  });

  return {
    email,
    password,
    rememberMe,
    errors: form.errors,
    serverError,
    isSubmitting: form.isSubmitting,
    submit: async (event?: Event): Promise<void> => {
      await submitHandler(event);
    },
  };
}
