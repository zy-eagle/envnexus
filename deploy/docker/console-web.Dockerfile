FROM node:20-alpine AS builder
WORKDIR /app

# 1. Install dependencies using pnpm and Taobao mirror
RUN npm install -g pnpm && \
    pnpm config set registry https://registry.npmmirror.com && \
    pnpm config set fetch-retries 5 && \
    pnpm config set fetch-retry-mintimeout 20000 && \
    pnpm config set fetch-retry-maxtimeout 120000

COPY apps/console-web/package.json apps/console-web/pnpm-lock.yaml ./
# Only install production dependencies first if possible, or just install all
RUN pnpm install --frozen-lockfile

# 2. Copy source code and build
COPY apps/console-web/ ./
RUN pnpm run build

# 3. Production image
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV production

COPY --from=builder /app/package.json ./
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public

EXPOSE 3000

CMD ["npm", "start"]
