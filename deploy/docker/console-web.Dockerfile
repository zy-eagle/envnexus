FROM node:20-alpine AS builder
WORKDIR /app

# Enable corepack for pnpm and set npm mirror
RUN corepack enable && \
    npm config set registry https://registry.npmmirror.com

# Copy package.json and lockfile first to leverage Docker cache
COPY apps/console-web/package.json apps/console-web/pnpm-lock.yaml ./

# Install dependencies using pnpm with mirror
RUN pnpm config set registry https://registry.npmmirror.com && \
    pnpm install --frozen-lockfile --prefer-offline

# Copy source code and build
COPY apps/console-web/ ./
# Disable Next.js telemetry to speed up build
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm run build

# Production image
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Copy standalone output and static files
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

CMD ["node", "server.js"]
