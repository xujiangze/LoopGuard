# Stage 1: 构建前端
FROM node:22-alpine AS frontend
RUN corepack enable
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# Stage 2: 构建 Go 二进制
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
COPY --from=frontend /app/web/dist/ ./web/dist/
COPY web/embed.go ./web/embed.go
RUN CGO_ENABLED=0 go build -o /loopguard ./cmd/loopguard

# Stage 3: 最小运行时
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /loopguard /usr/local/bin/loopguard
RUN mkdir -p /app/workspace
ENTRYPOINT ["loopguard"]
CMD ["serve"]
