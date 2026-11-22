import { expect, within } from '@storybook/test';
import type { Meta, StoryObj } from '@storybook/vue3';

import RuiButton from '../RuiButton/RuiButton.vue';
import RuiCard from './RuiCard.vue';
import RuiTextField from '../RuiTextField/RuiTextField.vue';

const meta = {
  title: 'UI/Card',
  component: RuiCard,
  tags: ['autodocs'],
} satisfies Meta<typeof RuiCard>;

export default meta;
type Story = StoryObj<typeof meta>;

export const CompositionForm: Story = {
  render: () => ({
    components: { RuiCard, RuiTextField, RuiButton },
    template: `
      <RuiCard title="Вход" style="width: 320px;">
        <div style="display: flex; flex-direction: column; gap: 12px;">
          <RuiTextField label="Email" type="email" placeholder="user@example.com" />
          <RuiTextField label="Пароль" type="password" autocomplete="current-password" />
          <RuiButton block>Войти</RuiButton>
        </div>
      </RuiCard>
    `,
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByLabelText('Email')).toBeInTheDocument();
    await expect(canvas.getByRole('button', { name: 'Войти' })).toBeInTheDocument();
  },
};
