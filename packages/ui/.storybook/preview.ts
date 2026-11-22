import type { Preview } from '@storybook/vue3-vite';

import '../src/tokens.css';

const preview: Preview = {
  parameters: {
    layout: 'centered',
  },
};

export default preview;
