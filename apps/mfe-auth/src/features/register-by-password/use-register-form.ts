/**
 * features/register-by-password — регистрация по email и паролю.
 */
import { ref, type Ref } from 'vue';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import * as z from 'zod';

import { SessionApiError, sessionApi, type SessionUser } from '../../entities/session/api';

/** zod-схема формы регистрации (password + confirm проверяются вместе). */
const registerSchema = z
  .object({
    name: z.string().max(100, 'Не больше 100 символов'),
    email: z
      .string()
      .min(1, 'Укажите email')
      .regex(/^[^\s@]+@[^\s@]+\.[^\s@]+$/, 'Некорректный email'),
    password: z
      .string()
      .min(8, 'Минимум 8 символов')
      .regex(/[A-Za-zА-Яа-я]/, 'Добавьте буквы')
      .regex(/\d/, 'Добавьте цифру'),
    confirm: z.string().min(1, 'Повторите пароль'),
  })
  .refine((v) => v.password === v.confirm, {
    message: 'Пароли не совпадают',
    path: ['confirm'],
  });

/**
 * Человекочитаемый текст доменных ошибок.
 *
 * @param err
 */
function mapAuthError(err: SessionApiError): string {
  switch (err.code) {
    case 'email_already_exists':
      return 'Пользователь с таким email уже зарегистрирован';
    case 'rate_limited':
      return 'Слишком много попыток — попробуйте позже';
    default:
      return err.message;
  }
}

export interface UseRegisterFormOptions {
  /** Успешная регистрация (авто-логин уже выполнен BFF). */
  onSuccess: (user: SessionUser) => void;
}

export interface RegisterForm {
  name: Ref<string | undefined>;
  email: Ref<string | undefined>;
  password: Ref<string | undefined>;
  confirm: Ref<string | undefined>;
  /** Ошибки полей (vee-validate). */
  errors: Ref<Partial<Record<'name' | 'email' | 'password' | 'confirm', string>>>;
  serverError: Ref<string>;
  isSubmitting: Ref<boolean>;
  submit: (event?: Event) => Promise<void>;
}

/**
 *
 * @param options
 */
export function useRegisterForm(options: UseRegisterFormOptions): RegisterForm {
  const serverError = ref('');

  const form = useForm({
    validationSchema: toTypedSchema(registerSchema),
    initialValues: { name: '', email: '', password: '', confirm: '' },
  });

  const name = form.defineField('name')[0];
  const email = form.defineField('email')[0];
  const password = form.defineField('password')[0];
  const confirm = form.defineField('confirm')[0];

  const submitHandler = form.handleSubmit(async (values) => {
    serverError.value = '';
    try {
      const user = await sessionApi.register(
        values.email,
        values.password,
        values.name || undefined,
      );
      options.onSuccess(user);
    } catch (err) {
      serverError.value =
        err instanceof SessionApiError
          ? mapAuthError(err)
          : 'Сервис авторизации недоступен, попробуйте позже';
    }
  });

  return {
    name,
    email,
    password,
    confirm,
    errors: form.errors,
    serverError,
    isSubmitting: form.isSubmitting,
    submit: async (event?: Event): Promise<void> => {
      await submitHandler(event);
    },
  };
}
