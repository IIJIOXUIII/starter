# Аналогично apps/mfe-auth/Dockerfile, собирает mfe-admin.
FROM node:24-alpine AS build
RUN corepack enable
WORKDIR /repo
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json turbo.json ./
COPY packages/ packages/
COPY apps/mfe-admin/ apps/mfe-admin/
RUN pnpm install --frozen-lockfile
RUN pnpm exec turbo run build --filter=@rosterra/mfe-admin

FROM nginx:1.27-alpine
COPY apps/mfe-admin/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /repo/apps/mfe-admin/dist /usr/share/nginx/html
EXPOSE 80
