import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vite';

// С M4 сюда добавляется Module Federation 2.0 (@module-federation/vite):
// exposes: UsersListPage, UserEditPage; shared: vue, pinia (singletons).
export default defineConfig({
  plugins: [vue()],
});
