import type { Meta, StoryObj } from '@storybook/vue3';

import RuiAlert from './RuiAlert.vue';

const meta = {
  title: 'UI/Alert',
  component: RuiAlert,
  tags: ['autodocs'],
  args: { variant: 'info' },
} satisfies Meta<typeof RuiAlert>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Info: Story = {
  render: (args) => ({
    components: { RuiAlert },
    setup: () => ({ args }),
    template: '<RuiAlert v-bind="args">Письмо с подтверждением отправлено на email.</RuiAlert>',
  }),
};

export const Error: Story = {
  args: { variant: 'error' },
  render: Info.render,
};

export const Success: Story = {
  args: { variant: 'success' },
  render: Info.render,
};
