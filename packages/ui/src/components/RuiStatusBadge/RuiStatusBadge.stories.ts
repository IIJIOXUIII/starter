import { expect, within } from '@storybook/test';
import type { Meta, StoryObj } from '@storybook/vue3';

import RuiStatusBadge from './RuiStatusBadge.vue';

const meta = {
  title: 'UI/StatusBadge',
  component: RuiStatusBadge,
  tags: ['autodocs'],
  args: { status: 'active' },
} satisfies Meta<typeof RuiStatusBadge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Active: Story = {};

export const Blocked: Story = {
  args: { status: 'blocked' },
};

export const Pending: Story = {
  args: { status: 'pending' },
};

export const CustomLabel: Story = {
  args: { status: 'pending', label: 'На модерации' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('На модерации')).toBeInTheDocument();
  },
};
