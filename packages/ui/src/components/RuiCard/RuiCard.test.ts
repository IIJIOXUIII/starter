import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import RuiCard from './RuiCard.vue';

describe('RuiCard', () => {
  it('рендерит слот без заголовка по умолчанию', () => {
    const wrapper = mount(RuiCard, { slots: { default: 'Контент' } });
    expect(wrapper.text()).toContain('Контент');
    expect(wrapper.find('h2').exists()).toBe(false);
  });

  it('показывает заголовок, если передан', () => {
    const wrapper = mount(RuiCard, { props: { title: 'Вход' }, slots: { default: 'Контент' } });
    expect(wrapper.find('h2').text()).toBe('Вход');
  });
});
