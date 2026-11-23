# MF-remote как статика: vite build → nginx. remoteEntry/mf-manifest — без кэша,
# ассеты — immutable. За ingress сервисы лежат на /mfe/auth/ и /mfe/admin/
# (same-origin: куки и MF-манифесты без CORS).
# Сборка: docker build -f apps/mfe-auth/Dockerfile -t rosterra/mfe-auth .
FROM node:24-alpine AS build
RUN corepack enable
WORKDIR /repo
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json turbo.json ./
COPY packages/ packages/
COPY apps/mfe-auth/ apps/mfe-auth/
RUN pnpm install --frozen-lockfile
RUN pnpm exec turbo run build --filter=@rosterra/mfe-auth

FROM nginx:1.27-alpine
COPY apps/mfe-auth/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /repo/apps/mfe-auth/dist /usr/share/nginx/html
EXPOSE 80
