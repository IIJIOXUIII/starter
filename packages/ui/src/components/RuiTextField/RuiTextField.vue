<script setup lang="ts">
import { computed, useId } from 'vue';

interface Props {
  /** Подпись поля. */
  label: string;
  /** Текущее значение (v-model). */
  modelValue?: string;
  /** Тип input: text/email/password. */
  type?: 'text' | 'email' | 'password';
  /** Подсказка внутри поля. */
  placeholder?: string;
  /** Сообщение об ошибке валидации (показывается и связывается с полем). */
  error?: string;
  /** Автозаполнение браузером. */
  autocomplete?: string;
  /** Блокирует ввод. */
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  type: 'text',
  placeholder: '',
  error: '',
  autocomplete: undefined,
  disabled: false,
});

const emit = defineEmits<{
  /** Значение изменилось (v-model). */
  'update:modelValue': [value: string];
  /** Поле потеряло фокус (для валидации on blur). */
  blur: [];
}>();

const inputId = useId();
const describedBy = computed(() => (props.error !== '' ? `${inputId}-error` : undefined));
</script>

<template>
  <div class="rui-text-field" :class="{ 'rui-text-field--invalid': error !== '' }">
    <label class="rui-text-field__label" :for="inputId">{{ label }}</label>
    <input
      :id="inputId"
      :type="type"
      class="rui-text-field__input"
      :value="modelValue"
      :placeholder="placeholder || undefined"
      :autocomplete="autocomplete"
      :disabled="disabled || undefined"
      :aria-invalid="error !== '' || undefined"
      :aria-describedby="describedBy"
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      @blur="emit('blur')"
    />
    <p v-if="error !== ''" :id="describedBy" class="rui-text-field__error" role="alert">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.rui-text-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.rui-text-field__label {
  font-size: var(--ui-font-size-sm);
  font-weight: 600;
  color: var(--ui-color-text);
}

.rui-text-field__input {
  height: 40px;
  padding: 0 var(--ui-space-md);
  border: 1px solid var(--ui-color-border);
  border-radius: var(--ui-radius-md);
  font-family: var(--ui-font-family);
  font-size: var(--ui-font-size-md);
  color: var(--ui-color-text);
  background-color: var(--ui-color-bg);
  transition:
    border-color 0.15s ease,
    box-shadow 0.15s ease;
}

.rui-text-field__input::placeholder {
  color: var(--ui-color-text-muted);
}

.rui-text-field__input:focus {
  outline: none;
  border-color: var(--ui-color-primary);
  box-shadow: var(--ui-focus-ring);
}

.rui-text-field__input:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.rui-text-field--invalid .rui-text-field__input {
  border-color: var(--ui-color-danger);
}

.rui-text-field__error {
  margin: 0;
  font-size: var(--ui-font-size-sm);
  color: var(--ui-color-danger);
}
</style>
