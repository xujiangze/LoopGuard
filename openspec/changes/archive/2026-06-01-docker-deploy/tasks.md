## 1. 前端嵌入 Go 二进制

- [x] 1.1 创建 `web/embed.go`，使用 `//go:embed dist/*` 将前端构建产物嵌入变量
- [x] 1.2 修改 `internal/api/router.go`，添加 `/assets` 静态文件路由和 `/` index.html 路由
- [x] 1.3 验证：本地 `cd web && pnpm build && cd .. && go build` 编译通过
- [x] 1.4 验证：启动服务后浏览器访问 `/` 能看到前端页面，`/api/v1/*` 正常

## 2. Dockerfile

- [x] 2.1 创建 `Dockerfile`，三阶段构建（node:22-alpine → golang:1.26-alpine → alpine:3.21）
- [ ] 2.2 验证：`docker build -t loopguard .` 构建成功
- [ ] 2.3 验证：`docker run --rm loopguard --help` 输出正常

## 3. Docker Compose

- [x] 3.1 创建 `docker-compose.yaml`，定义 loopguard + mysql 服务
- [x] 3.2 配置 MySQL named volume 持久化、内存限制、健康检查
- [x] 3.3 配置 loopguard 依赖 MySQL 健康检查、内存限制、环境变量
- [x] 3.4 创建 `.env.example` 包含所有部署变量及注释

## 4. 集成验证

- [ ] 4.1 验证：`docker compose up -d` 一键启动，两个容器均健康
- [ ] 4.2 验证：浏览器访问 loopguard 服务可看到前端 + API 正常
- [x] 4.3 运行 `go test ./...` 确保无回归
