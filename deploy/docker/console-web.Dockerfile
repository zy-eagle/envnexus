FROM node:20-alpine AS builder
WORKDIR /app

RUN corepack enable

COPY apps/console-web/package.json apps/console-web/pnpm-lock.yaml ./

RUN pnpm install --frozen-lockfile

COPY apps/console-web/ ./

# Next may omit public/; runner stage COPY requires the path to exist
RUN mkdir -p public

ARG API_PROXY_TARGET=http://platform-api:8080
ENV API_PROXY_TARGET=${API_PROXY_TARGET}
ENV NEXT_TELEMETRY_DISABLED=1

RUN pnpm run build

FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

CMD ["node", "server.js"]
