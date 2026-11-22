import { expect, within } from '@storybook/test';
import type { Meta, StoryObj } from '@storybook/vue3';

import RuiCard from './RuiCard.vue';

const meta = {
  title: 'UI/Card',
  component: RuiCard,
  tags: ['autodocs'],
  args: { title: '' },
} satisfies Meta<typeof RuiCard>;

export default meta;
type Story = StoryObj<typeof meta>;

export const WithoutTitle: Story = {
  render: (args) => ({
    components: { RuiCard },
    setup: () => ({ args }),
    template: '<RuiCard v-bind="args">Содержимое карточки.</RuiCard>',
  }),
};

export const WithTitle: Story = {
  args: { title: 'Вход в систему' },
  render: () => ({
    components: { RuiCard },
    template: '<RuiCard title="Вход в систему">Форма авторизации.</RuiCard>',
  }),
};

export const ContentCheck: Story = {
  args: { title: 'Проверка' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByText('Проверка')).toBeInTheDocument();
  },
};
