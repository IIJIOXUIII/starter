import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import RuiAlert from './RuiAlert.vue';

describe('RuiAlert', () => {
  it('по умолчанию variant=info и role=status', () => {
    const wrapper = mount(RuiAlert, { slots: { default: 'Текст' } });
    expect(wrapper.classes()).toContain('rui-alert--info');
    expect(wrapper.attributes('role')).toBe('status');
    expect(wrapper.text()).toContain('Текст');
  });

  it('variant=error даёт role=alert для скринридеров', () => {
    const wrapper = mount(RuiAlert, { props: { variant: 'error' }, slots: { default: 'Ошибка' } });
    expect(wrapper.classes()).toContain('rui-alert--error');
    expect(wrapper.attributes('role')).toBe('alert');
  });

  it('variant=success', () => {
    const wrapper = mount(RuiAlert, { props: { variant: 'success' }, slots: { default: 'Ок' } });
    expect(wrapper.classes()).toContain('rui-alert--success');
  });
});
