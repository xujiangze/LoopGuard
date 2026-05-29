# LoopGuard 设计文档

日期：2026-05-29
状态：已批准设计，待实现

## 1. 背景与目标

在使用 hermes-agent、openclaw 等自主 AI agent 落地时，遇到"人在闭环"缺失的困难。查询等简单操作 AI 可直接执行，但提交等危险操作需要引入人工审批卡点。

LoopGuard 是一个轻量的审批卡点平台：

- AI 拿到的 skill 是"提交任务"，提交动作本身不执行危险操作，而是创建一张**审批工单**。
- 平台托管一批 go 程序（二进制），AI 只能对**已注册**的程序提交执行请求。
- 工单创建时，平台自动用 `--only-print` 跑一次（dry-run），把"将要执行什么"展示在审批页。
- 必须由**指定审批人**在页面通过后，平台才去掉 `--only-print` 在**沙盒**中真正执行。
- AI 全程**轮询**工单状态接口获取执行结果。

第一期原则：简单卡点、不做侵入性强的功能，但在架构上为沙盒升级预留扩展点。

## 2. 关键设计决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 架构 | 单体服务 + 抽象 Executor | 第一期最快落地，Executor 接口预留沙盒升级点 |
| --only-print 责任模型 | 信任 + 登记校验 | 注册时声明支持，dry-run 阶段做冒烟校验挡掉未正确实现的程序 |
| 沙盒第一期 | 抽象接口 + 进程级实现先行 | Executor 接口预留 Docker，第一期用 os/exec + 进程组 |
| 程序托管 | 先注册白名单 | AI 只能提交已注册程序，可控、审批人可自动关联 |
| 用户系统 | 两角色（user / admin） | JWT 登录，审批动作可追踪是谁做的 |
| AI 身份 | API Key | 服务账号，可吊销轮转，适合自动化 agent |
| 执行结果回传 | AI 轮询状态 | 无需回调基础设施，最简 |
| dry-run 校验 | DRYRUN-OK 标记行 + 退出码 0 | 双重校验，挡掉作者忘实现 --only-print 的程序 |
| 进程隔离第一期 | os/exec + 进程组，不上 ulimit | 第一期任务量小，先不引入资源限制 |

## 3. 架构总览

```
┌─────────────┐   X-API-Key    ┌──────────────────────────────────────┐
│  AI agent   │ ─────────────▶ │            LoopGuard (Go/Gin)          │
│ (hermes...) │ ◀── 轮询状态 ── │  ┌──────────┐  ┌─────────┐  ┌────────┐│
└─────────────┘                │  │   API    │─▶│ Service │─▶│Executor││ ──▶ 被托管 go 程序
                               │  │ handlers │  │ 状态机   │  │(接口)   ││     (--only-print / 正式)
┌─────────────┐   JWT          │  └──────────┘  └─────────┘  └────────┘│
│  审批人/管理  │ ─────────────▶ │         │            │                │
│  (React UI) │ ◀── 工单数据 ── │         ▼            ▼                │
└─────────────┘                │      ┌─────────────────────┐         │
                               │      │       MySQL         │         │
                               │      └─────────────────────┘         │
                               └──────────────────────────────────────┘
```

单体二进制同时提供：AI 提交接口（API Key 认证）、人工审批接口（JWT）、管理接口（JWT + admin）。执行通过 `Executor` 接口完成，第一期注入 `ProcessExecutor`。

## 4. 工单状态机

```
[AI 提交]
   │
   ▼
PENDING_DRYRUN ──(自动跑 --only-print)──▶ PENDING_APPROVAL
   │                                            │
   │ (dry-run 校验失败)                          ├──(审批人拒绝)──▶ REJECTED
   ▼                                            │
DRYRUN_FAILED                                   ├──(审批人通过)──▶ APPROVED
                                                │                    │
                                                │                    ▼
                                                │               EXECUTING ──(去掉 --only-print 沙盒执行)
                                                │                    │
                                                │          ┌─────────┴─────────┐
                                                │          ▼                   ▼
                                                │        DONE              EXEC_FAILED
```

状态枚举：`pending_dryrun`、`dryrun_failed`、`pending_approval`、`approved`、`executing`、`done`、`exec_failed`、`rejected`。

**流程说明：**
1. AI 提交 → 工单进 `pending_dryrun`。
2. LoopGuard 自动用 `--only-print` 跑一次，输出存进 `dryrun_output`。
3. dry-run 校验（见 §7）通过 → `pending_approval`，返回审批 URL 给 AI；校验失败 → `dryrun_failed`，**不进入审批**。
4. AI 通知用户去审批；审批人在页面看到 dry-run 内容。
5. 审批人通过 → `approved` → `executing`，沙盒去掉 `--only-print` 真正执行 → `done` / `exec_failed`。审批人拒绝 → `rejected`。
6. AI 轮询 `GET /api/v1/tickets/{id}` 获取最终结果。

## 5. 数据模型（MySQL）

```sql
-- 用户表（两角色：user / admin）
CREATE TABLE users (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  username      VARCHAR(64)  NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,         -- bcrypt
  role          ENUM('user','admin') NOT NULL DEFAULT 'user',
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- API Key 表（AI 服务账号用，与 user 解耦，便于轮转/吊销）
CREATE TABLE api_keys (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  name        VARCHAR(64)  NOT NULL,           -- 例如 "hermes-agent"
  key_hash    VARCHAR(255) NOT NULL UNIQUE,    -- 只存哈希，明文创建时返回一次
  enabled     BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 已注册程序（白名单）
CREATE TABLE programs (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  project         VARCHAR(128) NOT NULL,       -- 项目名
  name            VARCHAR(128) NOT NULL,       -- 脚本/程序逻辑名，AI 用它引用
  binary_path     VARCHAR(512) NOT NULL,       -- 宿主机上二进制绝对路径
  help_text       TEXT,                        -- 注册时抓取的 --help 输出
  params_schema   JSON,                        -- 参数白名单/说明
  approver_id     BIGINT NOT NULL,             -- 默认审批人 -> users.id
  timeout_sec     INT NOT NULL DEFAULT 300,    -- 执行超时
  supports_dryrun BOOLEAN NOT NULL DEFAULT TRUE,-- 信任+登记校验：声明支持 --only-print
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_project_name (project, name),
  FOREIGN KEY (approver_id) REFERENCES users(id)
);

-- 工单
CREATE TABLE tickets (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  program_id    BIGINT NOT NULL,
  args          JSON NOT NULL,                 -- AI 提交的参数 {key:value}
  status        ENUM('pending_dryrun','dryrun_failed','pending_approval',
                     'approved','executing','done','exec_failed','rejected') NOT NULL,
  submitted_by  BIGINT NOT NULL,               -- api_keys.id（AI 服务账号）
  approver_id   BIGINT NOT NULL,               -- 快照自 programs.approver_id
  dryrun_output MEDIUMTEXT,                     -- --only-print 的输出，审批页展示
  approved_by   BIGINT,                         -- users.id，谁审批的（可追踪）
  approved_at   DATETIME,
  reject_reason VARCHAR(512),
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (program_id) REFERENCES programs(id),
  FOREIGN KEY (approver_id) REFERENCES users(id),
  KEY idx_status (status),
  KEY idx_approver (approver_id, status)
);

-- 执行记录（dry-run 和正式各一条）
CREATE TABLE executions (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  ticket_id   BIGINT NOT NULL,
  kind        ENUM('dryrun','real') NOT NULL,
  command     VARCHAR(2048) NOT NULL,          -- 实际拼出的命令行（脱敏后）
  exit_code   INT,
  stdout      MEDIUMTEXT,
  stderr      MEDIUMTEXT,
  duration_ms INT,
  started_at  DATETIME,
  finished_at DATETIME,
  FOREIGN KEY (ticket_id) REFERENCES tickets(id),
  KEY idx_ticket (ticket_id)
);
```

**设计要点：**
- `approver_id` 在工单创建时从 program **快照**一份，避免之后改注册影响历史单。
- API Key 只存哈希，创建时明文返回一次。
- `params_schema` 做参数白名单——AI 提交的参数 key 必须在白名单内，防止夹带 `--only-print` 之外的危险 flag。
- `supports_dryrun` 落实"信任+登记校验"：注册时显式声明支持 `--only-print`。

## 6. API 设计

认证分两套：AI 接口走 `X-API-Key` header；人工/管理接口走 `Authorization: Bearer <JWT>`。

### 6.1 AI 接口（X-API-Key）

```
POST /api/v1/tickets                       提交任务（创建工单）
  body: { project, name, args: {k:v} }
  → 校验程序已注册 & enabled → 校验 args key 在 params_schema 白名单内
  → 创建工单(pending_dryrun) → 跑 --only-print
  → 返回: { ticket_id, status, approval_url, next_action }

GET  /api/v1/tickets/{id}                  轮询工单状态
  → 返回: { status, dryrun_output, execution: {exit_code, stdout, stderr}, ... }
```

提交接口返回示例：

```json
{
  "ticket_id": 42,
  "status": "pending_approval",
  "approval_url": "https://loopguard.internal/tickets/42",
  "next_action": "任务已提交，需人工审批。请通知用户访问审批链接 https://loopguard.internal/tickets/42 找审批人 @zhangsan 审批。审批通过后任务将自动执行，你可轮询 GET /api/v1/tickets/42 获取结果。"
}
```

后端在 `next_action` 中主动给出指引文案，引导 AI 通知用户找人审批。

### 6.2 人工接口（JWT）

```
POST /api/v1/auth/login                    登录拿 JWT
GET  /api/v1/tickets?status=pending_approval&mine=true   我待审批的工单列表
GET  /api/v1/tickets/{id}                  工单详情（看 dry-run 输出）
POST /api/v1/tickets/{id}/approve          审批通过 → 触发沙盒执行
POST /api/v1/tickets/{id}/reject           驳回 body:{reason}
```

**安全要点**：`approve` 接口必须校验当前 JWT 用户 == 工单的 `approver_id`（或 admin），落实"必须由指定用户审批才能放行"。

### 6.3 管理接口（JWT, admin only）

```
POST /api/v1/programs                      注册程序
  body: { project, name, binary_path, approver_id, timeout_sec, params_schema }
  → 后端自动执行 binary_path --help 抓取 help_text 存档；探测 --only-print 是否被识别
GET  /api/v1/programs                       程序列表
PUT  /api/v1/programs/{id}                  改注册（启用/禁用/改审批人）
POST /api/v1/users                          建用户
POST /api/v1/api-keys                       建 AI 服务账号 Key（明文返回一次）
```

## 7. Executor 抽象 + --only-print 协议 + 沙盒

### 7.1 Executor 接口（预留扩展点）

```go
type ExecRequest struct {
    BinaryPath string
    Args       []string          // 已拼好的参数（不含/含 --only-print）
    DryRun     bool              // true 时追加 --only-print
    TimeoutSec int
    WorkDir    string            // 每个 ticket 独立工作目录
    Env        []string          // 白名单环境变量
}

type ExecResult struct {
    Command    string
    ExitCode   int
    Stdout     string
    Stderr     string
    DurationMs int64
    TimedOut   bool
}

type Executor interface {
    Run(ctx context.Context, req ExecRequest) (*ExecResult, error)
}
```

第一期实现 `ProcessExecutor`，后期可加 `DockerExecutor`，业务层只依赖接口。配置项 `executor.type=process|docker` 决定注入哪个。`WorkDir` / `Env` / `TimeoutSec` 三个字段为 Docker 化预留——`DockerExecutor` 只需翻译成 `docker run -v workdir -e env --stop-timeout`，业务层零改动。

### 7.2 ProcessExecutor 第一期隔离手段

1. **独立工作目录**：每个 ticket 在 `workspace/ticket-{id}/` 下执行，互不干扰，执行后保留供审计。
2. **环境变量白名单**：不继承宿主机全部环境，只注入 `PATH` 等必需项 + 程序注册时声明的白名单变量，避免泄露 secret。
3. **强制超时**：用 `context.WithTimeout` + 进程组 kill，超时整组杀掉防残留子进程。
4. **退出码捕获**：区分正常退出、非零退出、超时三种。

第一期**不上 ulimit**（资源限制），任务量上来后再考虑。

### 7.3 --only-print 协议（信任 + 登记校验）

三道关：

1. **注册关**：注册程序时 `supports_dryrun=true` 必须显式声明，后端自动探测 `--only-print` 参数是否被识别（不报 "unknown flag"）。不通过则拒绝注册。

2. **dry-run 冒烟校验**：每张工单正式执行前先跑 `--only-print`，校验规则（两者都必须满足）：
   - 退出码必须为 **0**（dry-run 不该失败）
   - 输出中必须包含约定的协议标记行 **`DRYRUN-OK`**（程序在 dry-run 分支首行打印，在注册文档中约定）
   
   任一不满足 → 判 `dryrun_failed`，**绝不进入审批**。这把"程序作者忘了实现 --only-print"的风险挡在审批之前。

3. **执行关**：审批通过后去掉 `--only-print` 重新拼参数执行，命令行落库 `executions.command` 供审计。

### 7.4 被托管程序的协议契约（约定）

被 LoopGuard 托管的 go 程序必须遵守：
- 支持 `--only-print` 参数；收到时**只打印将要执行的动作，绝不执行**。
- 在 `--only-print` 分支输出包含 `DRYRUN-OK` 标记行，且退出码为 0。
- 支持 `--help` 输出参数说明。

## 8. 用户系统（两角色）

- **登录**：用户名 + 密码（bcrypt），登录返回 JWT（含 user_id、role、过期时间）。
- **角色**：
  - `user`：能看「自己是审批人」的工单、审批/驳回这些工单。
  - `admin`：全部 user 权限 + 注册程序、管理用户、管理 API Key。
- **初始化**：首次启动用 CLI 子命令创建第一个 admin。
- **AI 身份**：不走用户表，用 api_keys。admin 在后台创建 Key，明文展示一次，交给 AI agent 配置。

## 9. 项目结构

```
LoopGuard/
├── cmd/loopguard/main.go          # 入口，含 --help / 子命令
├── internal/
│   ├── api/                       # Gin handlers + 路由 + 中间件(JWT/APIKey)
│   ├── service/                   # 业务逻辑：ticket 状态机、program 注册
│   ├── executor/                  # Executor 接口 + ProcessExecutor
│   ├── model/                     # GORM 模型
│   ├── store/                     # DB 访问层
│   └── config/                    # 配置加载
├── migrations/                    # SQL 迁移
├── web/                           # React 前端
└── docs/superpowers/specs/        # 设计文档
```

## 10. CLI（二进制 --help 可获取所有能力）

```
loopguard serve                    # 启动 HTTP 服务
loopguard migrate                  # 执行 DB 迁移
loopguard admin create-user        # 创建用户/管理员（首个 admin）
loopguard apikey create --name X   # 创建 AI 服务账号 Key
loopguard --help                   # 列出全部子命令
```

所有平台能力均通过二进制子命令暴露，可通过 `--help` 获取，符合"后台托管"运维要求。

## 11. 前端（React，最小页面集）

1. **登录页**
2. **待审批列表**（默认进来就是「我待审批的」）
3. **工单详情页**：展示程序信息、AI 提交的参数、dry-run 输出（含 `DRYRUN-OK` 段）、审批/驳回按钮、执行结果。
4. **管理页**（admin）：程序注册表单、用户管理、API Key 管理。

技术栈：React + 轻量 UI 库（如 Ant Design），调用 §6.2 / §6.3 的 JWT 接口。

## 12. 非目标（第一期不做）

- 队列 / 独立 Worker（任务量上来再从单体演进）
- Docker / gVisor 沙盒（接口已预留，第一期用进程级）
- 三角色 / 细粒度权限
- Webhook 回调（第一期 AI 轮询）
- ulimit 资源限制
- 多审批人 / 审批流编排
