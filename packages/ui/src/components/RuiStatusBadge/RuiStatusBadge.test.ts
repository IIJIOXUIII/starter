import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import RuiStatusBadge from './RuiStatusBadge.vue';

describe('RuiStatusBadge', () => {
  it('показывает человекочитаемое имя статуса по умолчанию', () => {
    const wrapper = mount(RuiStatusBadge, { props: { status: 'active' } });
    expect(wrapper.text()).toBe('Активен');
    expect(wrapper.classes()).toContain('rui-status-badge--active');
  });

  it('применяет цвет по статусу', () => {
    const wrapper = mount(RuiStatusBadge, { props: { status: 'blocked' } });
    expect(wrapper.classes()).toContain('rui-status-badge--blocked');
    expect(wrapper.text()).toBe('Заблокирован');
  });

  it('позволяет переопределить текст через label', () => {
    const wrapper = mount(RuiStatusBadge, { props: { status: 'pending', label: 'На модерации' } });
    expect(wrapper.text()).toBe('На модерации');
    expect(wrapper.classes()).toContain('rui-status-badge--pending');
  });
});
