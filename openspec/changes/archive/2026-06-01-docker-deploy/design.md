## Context

LoopGuard 目前没有容器化部署方案。Go 后端仅服务 API 路由，前端 SPA 需要独立部署。生产环境需要"单体二进制"体验——一个二进制包含前后端，加上 MySQL 即可运行。

用户机器内存限制 6G，需合理分配给 MySQL、Go 进程、executor 子进程。

## Goals / Non-Goals

**Goals:**
- Go 二进制嵌入前端静态文件，实现单体部署
- 多阶段 Dockerfile 构建最小镜像
- docker-compose.yaml 一键启动（app + MySQL）
- .env.example 降低部署门槛
- 内存预算控制：MySQL 1G + Go ~100M + Executor ~500M

**Non-Goals:**
- 不做 HTTPS/TLS 终止（留给外部反向代理或 ingress）
- 不做 Kubernetes 配置
- 不做 CI/CD 流水线集成
- 不做多节点/集群部署

## Decisions

### 1. 前端嵌入方式：go:embed

**选择**: 使用 `//go:embed` 将 `web/dist/` 目录嵌入二进制。

**理由**: 编译时打包，无运行时依赖，真正的单体二进制。前端使用 HashRouter，无需服务端路由 fallback。

**备选**: 运行时读取文件系统 → 需要额外目录挂载，破坏"单文件"体验。

### 2. Dockerfile 多阶段构建

```
Stage 1: node:22-alpine     → pnpm install + build → dist/
Stage 2: golang:1.26-alpine → COPY dist/ + go build
Stage 3: alpine:3.21        → 只拷贝二进制
```

**选择**: 三阶段构建。

**理由**: 最终镜像只包含二进制 + 最小运行时，预计 ~20-30M。Alpine 兼容 CGO（musl），Go 默认静态链接无需特殊处理。

### 3. docker-compose 服务划分

**选择**: 2 个服务（loopguard + mysql），单网络。

**理由**: 最简拓扑。前端已嵌入 Go 二进制，无需 nginx。MySQL 用官方镜像 + volume 持久化。

### 4. MySQL 内存配置

**选择**: `innodb_buffer_pool_size=256M`，容器 memory limit 1G。

**理由**: LoopGuard 是低并发审批平台，256M buffer pool 绑绑有余。1G limit 留足余量给连接和临时表。

## Risks / Trade-offs

- **[Embed 增加二进制体积]** → 前端 dist 约 2-3M，可忽略
- **[Alpine musl 与 CGO]** → Go 默认禁用 CGO 时纯静态链接，无兼容问题；MySQL driver 纯 Go 实现，无 CGO 依赖
- **[MySQL 数据持久化]** → 使用 named volume，`docker compose down` 不删除数据；需文档说明 `down -v` 会清数据
- **[Executor 子进程在容器中运行]** → 需挂载 workspace 目录；进程组 kill 在容器中正常工作
