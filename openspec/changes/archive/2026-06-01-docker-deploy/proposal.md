## Why

项目缺少容器化部署方案，需要 Dockerfile 和 docker-compose.yaml 来简化部署流程。同时 Go 后端目前不服务前端静态文件，生产环境无法以"单体二进制"方式运行。

## What Changes

- 添加 Go embed 支持，将前端 build 产物嵌入 Go 二进制
- Gin 路由增加前端静态文件服务（`/assets/*` 和 `/` → index.html）
- 添加多阶段 Dockerfile（node build → go build → minimal runtime）
- 添加 docker-compose.yaml（app + MySQL 两个容器）
- 添加 .env.example 部署配置模板

## Capabilities

### New Capabilities
- `docker-deploy`: 容器化部署支持，包括多阶段构建、docker-compose 编排、内存限制配置
- `frontend-embed`: 前端静态文件嵌入 Go 二进制，实现真正的单体部署

### Modified Capabilities

(无已有 spec 需要修改)

## Impact

- **Go 代码**: `internal/api/router.go` 需添加静态文件路由；新增 `web/embed.go` 处理 embed
- **构建流程**: Dockerfile 多阶段构建（node:22-alpine → golang:1.26-alpine → alpine:3.21）
- **依赖**: 无新依赖引入
- **部署**: 从"手动编译 + 配置 MySQL"变为 `docker compose up -d` 一键部署
- **内存**: 总预算 6G（MySQL 1G + Go ~100M + Executor ~500M + 系统余量）
