# CLAUDE.md

> AI 危险操作人工审批卡点平台 -- LoopGuard

## 项目愿景

LoopGuard 解决自主 AI agent 落地时"人在闭环"缺失的问题。AI 可提交工单但不能直接执行危险操作；平台自动 dry-run 校验后由指定审批人放行，审批通过才在沙盒中真正执行。

核心协议：被托管程序必须支持 `--only-print` 参数（不执行真实操作，只打印意图）并在 dry-run 输出中包含 `DRYRUN-OK` 标记。

## 架构总览

```
AI Agent (API Key) ──POST /tickets──> Gin Router ──> TicketService ──> Executor ──> 被托管程序
                                       |                                    (接口)
审批人 (JWT) ────────POST /approve──>  |                              ProcessExecutor
                                       v
                                    MySQL (GORM)
                                       ^
管理页 (JWT+admin) ──CRUD programs ──> |
```

**单体二进制**，Go 后端（Gin + GORM/MySQL）+ React 前端（Vite + Tailwind）。CLI 入口（Cobra）暴露 serve / migrate / admin / apikey 四个子命令。

### 请求处理流

1. AI 提交工单 -> 创建 ticket（pending_dryrun）-> 自动跑 `--only-print` dry-run
2. dry-run 通过（退出码 0 + stdout 含 `DRYRUN-OK`）-> pending_approval，返回 approval_url
3. dry-run 失败 -> dryrun_failed，不进入审批
4. 审批人批准 -> approved -> executing（去掉 `--only-print` 真正执行）-> done / exec_failed
5. 审批人驳回 -> rejected
6. AI 轮询 GET /tickets/:id 获取结果

### 工单状态机

```
pending_dryrun --> pending_approval --> approved --> executing --> done
       |                                      |
       v                                      v
  dryrun_failed                          exec_failed

pending_approval --> rejected
```

状态转换规则定义在 `internal/model/status.go` 的 `CanTransition()` 函数。

## 构建与开发

### 后端（Go）

```bash
# 编译
go build -o loopguard ./cmd/loopguard

# 运行测试（全部，SQLite 内存库）
go test ./...

# 运行测试（带覆盖率）
go test -cover ./...

# 启动服务（需 MySQL）
./loopguard serve

# 数据库迁移
./loopguard migrate

# 创建首个管理员
./loopguard admin create-user --username root --password xxx --admin

# 创建 AI 服务账号 Key
./loopguard apikey create --name my-agent
```

环境变量（全部 `LOOPGUARD_` 前缀）：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LOOPGUARD_HTTP_ADDR` | `:8080` | HTTP 监听地址 |
| `LOOPGUARD_DB_DSN` | (空，必填) | MySQL DSN |
| `LOOPGUARD_JWT_SECRET` | `dev-insecure-secret-change-me` | JWT 签名密钥 |
| `LOOPGUARD_BASE_URL` | `http://localhost:8080` | 外部地址，拼审批链接用 |
| `LOOPGUARD_EXECUTOR_TYPE` | `process` | 执行器类型 |
| `LOOPGUARD_WORKSPACE_DIR` | `./workspace` | 工作目录 |

### 前端（React/TypeScript）

```bash
cd web

# 安装依赖
pnpm install

# 开发（Vite dev server，代理 /api -> localhost:8080）
pnpm dev

# 构建
pnpm build

# Lint
pnpm lint
```

前端使用 HashRouter，Vite 开发时通过 `server.proxy` 将 `/api` 请求代理到后端 `:8080`。

## 测试策略

### Go 后端测试

- **测试框架**: `testify`（assert + require）
- **数据库**: 测试用 SQLite 内存库（`gorm.io/driver/sqlite`），不依赖外部 MySQL
- **Executor Mock**: service 层测试使用 `fakeExecutor` 实现 `executor.Executor` 接口
- **测试覆盖**: 每个包都有 `*_test.go`，覆盖 store、service、api handler、middleware、auth、model、config、executor、cli 各层
- **运行**: `go test ./...` 即可跑完全部测试

### 前端测试

- **E2E**: Playwright（`web/playwright.config.ts`），测试目录 `web/e2e/`（目前为空）
- **无单元测试**: 前端暂未配置 Vitest/Jest

## 代码结构

```
LoopGuard/
  cmd/loopguard/main.go          # 入口：Cobra 根命令 + 4 个子命令
  internal/
    api/                          # HTTP 层
      router.go                   # 路由注册 + 依赖注入 (Deps struct)
      middleware.go                # APIKeyAuth / JWTAuth / AdminOnly
      ai_handler.go               # AI 接口（submit, get）
      human_handler.go            # 审批人接口（login, list, approve, reject）
      admin_handler.go            # 管理接口（CRUD programs/users/apikeys）
    service/                      # 业务逻辑
      ticket.go                   # 工单提交 + 审批 + 驳回 + 参数校验/拼装
      program.go                  # 程序注册（含 --help 探测）
      dryrun.go                   # dry-run 校验（DRYRUN-OK 标记 + 退出码）
    executor/                     # 执行抽象
      executor.go                 # Executor 接口 + ExecRequest/ExecResult
      process.go                  # ProcessExecutor（os/exec + 进程组 kill）
    model/                        # GORM 模型
      models.go                   # User / APIKey / Program / Ticket / Execution
      status.go                   # TicketStatus 枚举 + 状态转换规则
    store/                        # 数据访问层
      store.go                    # 全部 CRUD 方法
    auth/                         # 认证工具
      jwt.go                      # JWT 签发/解析
      apikey.go                   # API Key 生成(lg_ 前缀) + SHA256 哈希
      password.go                 # bcrypt 哈希/校验
    config/                       # 配置
      config.go                   # 环境变量加载，全 LOOPGUARD_ 前缀
    cli/                          # CLI 子命令
      serve.go                    # serve 命令
      migrate.go                  # migrate 命令
      admin.go                    # admin create-user 命令
      apikey.go                   # apikey create 命令
      db.go                       # openStore（MySQL 连接）
  web/                            # React 前端（Vite + Tailwind + shadcn/ui）
    src/
      App.tsx                     # 路由定义（HashRouter）
      hooks/useAuth.tsx           # AuthContext + Provider
      lib/api.ts                  # fetch 封装（JWT 自动注入，401 自动跳登录）
      lib/auth.ts                 # localStorage token/user 管理
      types/index.ts              # TypeScript 类型定义（与 Go model 对应）
      components/Layout.tsx       # 侧边栏布局（普通用户/管理员菜单分离）
      components/ProtectedRoute.tsx  # JWT 登录守卫
      components/AdminRoute.tsx   # admin 角色守卫
      pages/                      # 页面组件
      components/ui/              # shadcn/ui 组件库
  docs/guide.md                   # 完整使用导引（API curl 示例）
  docs/superpowers/specs/         # 设计文档
```

## 关键设计决策

1. **两套认证体系**: AI 用 API Key（`X-API-Key` header，只存 SHA256 哈希），人类用 JWT（`Authorization: Bearer`），管理接口叠加 AdminOnly 中间件。
2. **Executor 接口预留扩展**: 当前仅 `ProcessExecutor`（os/exec + 进程组 kill + context 超时），接口设计已为 Docker 沙盒预留 WorkDir/Env/TimeoutSec 字段。
3. **参数白名单防注入**: AI 提交的 args key 必须在 Program 的 `params_schema` JSON 内；`only-print` 为系统保留字禁止传入。
4. **approver_id 快照**: 工单创建时从 Program 快照 approver_id，后续改注册不影响历史单的审批权限。
5. **Admin 可越权审批**: admin 角色在审批时使用工单自身的 approver_id 而非自身 user_id，实现管理代审。
6. **API Key 与 User 解耦**: API Key 独立建表（不挂 users），便于轮转/吊销，`submitted_by` 存的是 api_keys.id。

## API 路由速查

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| POST | /api/v1/auth/login | 无 | 登录获取 JWT |
| POST | /api/v1/tickets | API Key | AI 提交工单 |
| GET | /api/v1/tickets/:id | API Key 或 JWT | 轮询/查看工单 |
| GET | /api/v1/tickets/:id/executions | JWT | 查看执行记录 |
| GET | /api/v1/tickets | JWT | 我的待审批列表（?status= 筛选） |
| POST | /api/v1/tickets/:id/approve | JWT | 审批通过（触发真实执行） |
| POST | /api/v1/tickets/:id/reject | JWT | 驳回 |
| POST | /api/v1/programs | JWT+admin | 注册程序 |
| GET | /api/v1/programs | JWT+admin | 程序列表 |
| PUT | /api/v1/programs/:id | JWT+admin | 更新程序 |
| POST | /api/v1/users | JWT+admin | 创建用户 |
| GET | /api/v1/users | JWT+admin | 用户列表 |
| PUT | /api/v1/users/:id/password | JWT+admin | 重置密码 |
| POST | /api/v1/api-keys | JWT+admin | 创建 API Key（明文只返回一次） |
| GET | /api/v1/api-keys | JWT+admin | API Key 列表 |
| PUT | /api/v1/api-keys/:id | JWT+admin | 启用/禁用 API Key |
| DELETE | /api/v1/api-keys/:id | JWT+admin | 删除 API Key |

## 编码规范

- Go 代码遵循标准 Go 风格；`internal/` 下按职责分层（api -> service -> store -> model）
- 错误处理使用 `errors.New` / `fmt.Errorf`，handler 层统一转 HTTP 状态码
- 中文错误信息面向用户（直接返回给前端/AI），英文用于日志
- 前端使用 `@/` 路径别名指向 `web/src/`
- 前端 API 层封装在 `web/src/lib/api.ts`，所有请求走 `api.get/post/put/del`
- GORM AutoMigrate 管理表结构，无独立 SQL 迁移文件
- vendor 目录提交（`go mod vendor`），不依赖网络下载依赖

## 前端路由

| 路径 | 组件 | 权限 |
|---|---|---|
| /login | LoginPage | 公开 |
| / | TicketListPage | 登录用户 |
| /tickets/:id | TicketDetailPage | 登录用户 |
| /admin/programs | ProgramPage | admin |
| /admin/users | UserPage | admin |
| /admin/api-keys | ApiKeyPage | admin |

使用 HashRouter，URL 格式为 `http://host/#/path`。

## 安全要点

- API Key 明文仅在创建时返回一次，数据库只存 SHA256 哈希
- JWT 使用 HMAC-SHA256，有效期 12 小时
- 密码使用 bcrypt 哈希
- 执行命令时使用独立进程组（`Setpgid: true`），超时 kill 整组防残留子进程
- CORS 开发环境全开放（`AllowOriginFunc` 返回 true），生产环境需收紧
