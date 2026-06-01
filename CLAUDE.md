# CLAUDE.md

> AI 危险操作人工审批卡点平台 -- LoopGuard

## 项目愿景

LoopGuard 解决自主 AI agent 落地时"人在闭环"缺失的问题。AI 可提交工单但不能直接执行危险操作；平台自动 dry-run 校验后由指定审批人放行，审批通过才在沙盒中真正执行。

核心协议：被托管程序必须支持 `--only-print` 参数（不执行真实操作，只打印意图）并在 dry-run 输出中包含 `DRYRUN-OK` 标记。

## 技术栈

- **后端**: Go 1.26, Gin, GORM (MySQL/SQLite), Cobra CLI, JWT 认证
- **前端**: React 19, TypeScript, Vite, Tailwind CSS v4, shadcn/ui (base-nova), React Router v7
- **构建**: pnpm (前端), Go modules with vendor (后端), 前端通过 `go:embed` 嵌入二进制
- **部署**: Docker Compose (backend + frontend/nginx + MySQL), 也支持单二进制 `loopguard serve`（嵌入前端）

## 构建 & 开发命令

```bash
# 后端 — 需要先构建前端，因为 go:embed dist/*
cd web && pnpm install && pnpm build && cd ..
go build -o loopguard ./cmd/loopguard

# 开发模式 — 前端 dev server 反代 API 到后端 :8080
cd web && pnpm dev          # 前端 :5173，自动代理 /api → localhost:8080
LOOPGUARD_DB_DSN="file:test.db" go run ./cmd/loopguard serve  # 后端 :8080

# 测试
go test ./...                          # 全部后端测试
go test ./internal/service/...         # 单包测试
go test -run TestSubmit ./internal/api/ # 单个测试

# 前端测试
cd web && npx playwright test          # E2E 测试

# Docker 部署
cp .env.example .env && vim .env       # 配置环境变量
docker compose up -d                   # 启动全部服务
```

## 架构概览

```
AI Agent ──(API Key)──→ POST /api/v1/tickets ──→ dry-run ──→ 等待审批
                                                        │
审批人 ──(JWT 登录)──→ POST /tickets/:id/approve ──────┘──→ 真正执行
```

**工单状态机**: `pending_dryrun → pending_approval → approved → executing → done`
                                        ↘ dryrun_failed              ↘ exec_failed
                    pending_approval → rejected

**认证双轨制**: API Key（AI 提交工单用） + JWT（人类审批/管理用），Admin 角色独享管理接口。

## 后端结构 (`internal/`)

| 包 | 职责 |
|---|---|
| `config` | 环境变量配置加载（`LOOPGUARD_*`） |
| `model` | GORM 模型：User, APIKey, Program, Ticket, Execution + 工单状态机 |
| `store` | GORM 数据访问层，薄封装 CRUD |
| `service` | 核心业务逻辑：TicketService（提交/dry-run/审批/执行）、ProgramService（注册/更新/文件上传） |
| `executor` | `Executor` 接口 + ProcessExecutor（子进程执行，加 `--only-print` 做 dry-run） |
| `auth` | JWT 生成/验证、API Key HMAC 校验、bcrypt 密码哈希 |
| `api` | Gin 路由 + Handler（ai_handler: AI 侧、human_handler: 审批侧、admin_handler: 管理侧） |
| `cli` | Cobra 子命令：`serve`, `migrate`, `admin create-user`, `apikey` |

**关键数据流**:
1. AI 提交工单 → `ai_handler.Submit` → `TicketService.Submit` → 自动 dry-run → 状态变为 `pending_approval` 或 `dryrun_failed`
2. 审批人批准 → `human_handler.Approve` → `TicketService.Approve` → 立即执行真实命令 → 状态变为 `done` 或 `exec_failed`
3. 程序注册 → `admin_handler.CreateProgram` → `ProgramService.Register` → 上传文件 + 验证 `--help` 识别 `--only-print`

## 前端结构 (`web/src/`)

- `pages/`: LoginPage, TicketListPage, TicketDetailPage, ProgramPage, UserPage, ApiKeyPage
- `components/`: Layout, ProtectedRoute, AdminRoute, ui/ (shadcn)
- `hooks/useAuth.tsx`: 认证上下文
- `lib/api.ts`: 后端 API 调用封装
- `types/index.ts`: TypeScript 类型定义

## 环境变量

| 变量 | 说明 | 默认值 |
|---|---|---|
| `LOOPGUARD_HTTP_ADDR` | HTTP 监听地址 | `:8080` |
| `LOOPGUARD_DB_DSN` | 数据库连接串 | (必填) |
| `LOOPGUARD_JWT_SECRET` | JWT 签名密钥 | `dev-insecure-secret-change-me` |
| `LOOPGUARD_BASE_URL` | 外部访问地址（拼接审批链接） | `http://localhost:8080` |
| `LOOPGUARD_EXECUTOR_TYPE` | 执行器类型 | `process` |
| `LOOPGUARD_WORKSPACE_DIR` | 程序文件存放目录 | `./workspace` |

开发时 SQLite DSN: `file:test.db`，生产 MySQL DSN: `user:pass@tcp(host:3306)/db?charset=utf8mb4&parseTime=True&loc=Local`

## 部署方式

**Docker Compose (推荐)**: `docker compose up -d` 启动三容器 — backend (Go + 嵌入前端)、frontend (Nginx 反代 + 静态文件)、MySQL 8.0。Nginx 监听宿主机 `${LOOPGUARD_HTTP_PORT:-80}`，`/api/` 反代到 backend:8080。

**单二进制模式**: `loopguard serve` 单进程运行，前端通过 `go:embed` 嵌入，直接在 :8080 同时提供 API 和静态文件。

首次部署需执行 `loopguard admin create-user --username admin --password xxx --admin` 创建管理员。
