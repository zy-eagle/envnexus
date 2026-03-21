FROM node:20-alpine AS builder
WORKDIR /app
# Placeholder for Next.js build
RUN echo "console-web build placeholder" > index.html

FROM nginx:alpine
COPY --from=builder /app/index.html /usr/share/nginx/html/
EXPOSE 3000
# Placeholder CMD
CMD ["nginx", "-g", "daemon off;"]
