import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import RuiButton from './RuiButton.vue';

describe('RuiButton', () => {
  it('рендерит слот с текстом', () => {
    const wrapper = mount(RuiButton, { slots: { default: 'Сохранить' } });
    expect(wrapper.text()).toBe('Сохранить');
  });

  it('по умолчанию — primary/md/button', () => {
    const wrapper = mount(RuiButton);
    expect(wrapper.classes()).toContain('rui-button--primary');
    expect(wrapper.classes()).toContain('rui-button--md');
    expect(wrapper.attributes('type')).toBe('button');
  });

  it('применяет переданные variant, size и block', () => {
    const wrapper = mount(RuiButton, { props: { variant: 'danger', size: 'lg', block: true } });
    expect(wrapper.classes()).toContain('rui-button--danger');
    expect(wrapper.classes()).toContain('rui-button--lg');
    expect(wrapper.classes()).toContain('rui-button--block');
  });

  it('эмитит click при клике', async () => {
    const wrapper = mount(RuiButton);
    await wrapper.trigger('click');
    expect(wrapper.emitted('click')).toHaveLength(1);
  });

  it('при disabled не эмитит click и ставит атрибут disabled', async () => {
    const wrapper = mount(RuiButton, { props: { disabled: true } });
    expect(wrapper.attributes('disabled')).toBeDefined();
    await wrapper.trigger('click');
    expect(wrapper.emitted('click')).toBeUndefined();
  });

  it('пробрасывает HTML-тип submit', () => {
    const wrapper = mount(RuiButton, { props: { type: 'submit' } });
    expect(wrapper.attributes('type')).toBe('submit');
  });
});
