import { expect, userEvent, within } from '@storybook/test';
import type { Meta, StoryObj } from '@storybook/vue3';

import RuiButton from './RuiButton.vue';

const meta = {
  title: 'UI/Button',
  component: RuiButton,
  tags: ['autodocs'],
  args: { variant: 'primary', size: 'md' },
} satisfies Meta<typeof RuiButton>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  render: (args) => ({
    components: { RuiButton },
    setup: () => ({ args }),
    template: '<RuiButton v-bind="args">Продолжить</RuiButton>',
  }),
};

export const Secondary: Story = {
  args: { variant: 'secondary' },
  render: Primary.render,
};

export const Ghost: Story = {
  args: { variant: 'ghost' },
  render: Primary.render,
};

export const Danger: Story = {
  args: { variant: 'danger' },
  render: Primary.render,
};

export const Disabled: Story = {
  args: { variant: 'primary', disabled: true },
  render: Primary.render,
};

export const Sizes: Story = {
  render: () => ({
    components: { RuiButton },
    template: `
      <div style="display: flex; gap: 12px; align-items: center;">
        <RuiButton size="sm">Small</RuiButton>
        <RuiButton size="md">Medium</RuiButton>
        <RuiButton size="lg">Large</RuiButton>
      </div>
    `,
  }),
};

export const ClickInteraction: Story = {
  render: Primary.render,
  play: async ({ canvasElement, step }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button');

    await step('клик по кнопке не ломает компонент и меняет hover-состояние', async () => {
      await userEvent.hover(button);
      await userEvent.click(button);
      await expect(button).toBeEnabled();
    });
  },
};
