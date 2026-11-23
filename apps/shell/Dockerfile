# Shell/BFF (Nuxt 4, nitro node-server): сборка монорепо turbo'м, рантайм —
# только pruned node_modules + .output.
# Сборка: docker build -f apps/shell/Dockerfile -t rosterra/shell .
FROM node:24-alpine AS build
RUN corepack enable
WORKDIR /repo
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json turbo.json ./
COPY packages/ packages/
COPY apps/shell/ apps/shell/
COPY apps/mfe-auth/ apps/mfe-auth/
COPY apps/mfe-admin/ apps/mfe-admin/
RUN pnpm install --frozen-lockfile
RUN pnpm exec turbo run build --filter=@rosterra/shell
# Изолированные prod-зависимости shell'а (workspace-пакеты уже собраны turbo'м).
RUN pnpm --filter @rosterra/shell deploy --prod --ignore-scripts /out

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV NUXT_HOST=0.0.0.0
ENV NUXT_PORT=3000
RUN addgroup -g 10001 app && adduser -D -u 10001 -G app app
COPY --from=build /out/package.json ./
COPY --from=build /out/node_modules ./node_modules
COPY --from=build /repo/apps/shell/.output ./.output
USER app
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
