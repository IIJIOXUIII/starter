import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import RuiTextField from './RuiTextField.vue';

describe('RuiTextField', () => {
  it('связывает label с input через id и показывает подпись', () => {
    const wrapper = mount(RuiTextField, { props: { label: 'Email' } });
    const label = wrapper.find('label');
    const input = wrapper.find('input');
    expect(label.text()).toBe('Email');
    expect(label.attributes('for')).toBe(input.attributes('id'));
  });

  it('рендерит значение и тип по умолчанию text', () => {
    const wrapper = mount(RuiTextField, { props: { label: 'Email', modelValue: 'a@b.dev' } });
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('a@b.dev');
    expect(wrapper.find('input').attributes('type')).toBe('text');
  });

  it('пробрасывает type=password и autocomplete', () => {
    const wrapper = mount(RuiTextField, {
      props: { label: 'Пароль', type: 'password', autocomplete: 'current-password' },
    });
    expect(wrapper.find('input').attributes('type')).toBe('password');
    expect(wrapper.find('input').attributes('autocomplete')).toBe('current-password');
  });

  it('эмитит update:modelValue при вводе', async () => {
    const wrapper = mount(RuiTextField, { props: { label: 'Email' } });
    await wrapper.find('input').setValue('new@b.dev');
    expect(wrapper.emitted('update:modelValue')![0]).toEqual(['new@b.dev']);
  });

  it('эмитит blur при потере фокуса', async () => {
    const wrapper = mount(RuiTextField, { props: { label: 'Email' } });
    await wrapper.find('input').trigger('blur');
    expect(wrapper.emitted('blur')).toHaveLength(1);
  });

  it('показывает ошибку, aria-invalid и aria-describedby', () => {
    const wrapper = mount(RuiTextField, { props: { label: 'Email', error: 'Некорректный email' } });
    const input = wrapper.find('input');
    const errorEl = wrapper.find('.rui-text-field__error');
    expect(errorEl.text()).toBe('Некорректный email');
    expect(input.attributes('aria-invalid')).toBe('true');
    expect(input.attributes('aria-describedby')).toBe(errorEl.attributes('id'));
    expect(errorEl.attributes('role')).toBe('alert');
  });

  it('disabled блокирует ввод', () => {
    const wrapper = mount(RuiTextField, { props: { label: 'Email', disabled: true } });
    expect(wrapper.find('input').attributes('disabled')).toBeDefined();
  });
});
