import { expect, userEvent, within } from '@storybook/test';
import type { Meta, StoryObj } from '@storybook/vue3';

import RuiTextField from './RuiTextField.vue';

const meta = {
  title: 'UI/TextField',
  component: RuiTextField,
  tags: ['autodocs'],
  args: { label: 'Email', placeholder: 'user@example.com' },
} satisfies Meta<typeof RuiTextField>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Password: Story = {
  args: { label: 'Пароль', type: 'password', autocomplete: 'current-password', placeholder: '' },
};

export const WithError: Story = {
  args: { label: 'Email', modelValue: 'not-an-email', error: 'Некорректный email' },
};

export const Disabled: Story = {
  args: { label: 'Email', modelValue: 'locked@rosterra.dev', disabled: true },
};

export const TypingInteraction: Story = {
  play: async ({ canvasElement, step }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByLabelText('Email');

    await step('ввод текста попадает в поле', async () => {
      await userEvent.type(input, 'user@example.com');
      await expect(input).toHaveValue('user@example.com');
    });

    await step('после blur поле теряет фокус', async () => {
      await userEvent.click(input);
      await userEvent.tab();
      await expect(document.activeElement?.tagName).not.toBe('INPUT');
    });
  },
};

export const ErrorInteraction: Story = {
  args: { label: 'Email', error: 'Обязательное поле' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByLabelText('Email');
    const error = canvas.getByRole('alert');
    await expect(input).toHaveAttribute('aria-invalid', 'true');
    await expect(input).toHaveAttribute('aria-describedby', await error.getAttribute('id'));
  },
};
