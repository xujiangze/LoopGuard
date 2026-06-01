# LoopGuard 使用导引

LoopGuard 是 AI agent 危险操作的人工审批卡点服务。AI 通过 API Key 提交工单，系统自动执行 `--only-print` dry-run
校验，通过后由指定审批人放行，最终沙盒执行。

## 快速开始

### 1. 编译

```bash
go build -o loopguard ./cmd/loopguard
```

### 2. 准备 MySQL

```bash
mysql -u root -e "CREATE DATABASE loopguard CHARACTER SET utf8mb4;"
```

### 3. 配置环境变量

```bash
export LOOPGUARD_DB_DSN="root:@tcp(127.0.0.1:3306)/loopguard?parseTime=true"
export LOOPGUARD_JWT_SECRET="your-random-secret-here"
export LOOPGUARD_BASE_URL="http://localhost:8080"
```

| 环境变量                      | 默认值                             | 说明                   |
|---------------------------|---------------------------------|----------------------|
| `LOOPGUARD_HTTP_ADDR`     | `:8080`                         | HTTP 监听地址            |
| `LOOPGUARD_DB_DSN`        | _(空)_                           | MySQL DSN，必填         |
| `LOOPGUARD_JWT_SECRET`    | `dev-insecure-secret-change-me` | JWT 签名密钥，生产环境务必修改    |
| `LOOPGUARD_BASE_URL`      | `http://localhost:8080`         | 服务外部地址，用于拼审批链接       |
| `LOOPGUARD_EXECUTOR_TYPE` | `process`                       | 执行器类型（当前仅 `process`） |
| `LOOPGUARD_WORKSPACE_DIR` | `./workspace`                   | 工作目录                 |

### 4. 初始化

```bash
# 数据库迁移
./loopguard migrate

# 创建首个管理员
./loopguard admin create-user --username root --password admin123 --admin

# 创建 AI 服务账号 Key（明文只显示一次，请保存）
./loopguard apikey create --name my-agent
```

### 5. 启动服务

```bash
./loopguard serve
```

---

## CLI 命令参考

```
loopguard              # 查看所有子命令
loopguard serve        # 启动 HTTP 服务（自动迁移）
loopguard migrate      # 执行数据库迁移
loopguard admin        # 用户管理
  admin create-user    # 创建用户（--admin 设为管理员）
loopguard apikey       # API Key 管理
  apikey create        # 创建 Key（--name 指定名称，明文只显示一次）
```

---

## API 接口总览

所有接口前缀 `/api/v1`。认证方式分三种：

| 角色       | 认证方式           | Header                          |
|----------|----------------|---------------------------------|
| AI Agent | API Key        | `X-API-Key: lg_xxxxxxxx`        |
| 人类审批人    | JWT            | `Authorization: Bearer <token>` |
| 管理员      | JWT + admin 角色 | `Authorization: Bearer <token>` |

---

## 典型使用流程

```
管理员                        AI Agent                     审批人
  │                              │                           │
  │  1. 注册程序                  │                           │
  │  POST /programs              │                           │
  │──────────────────►           │                           │
  │                              │                           │
  │                              │  2. 提交工单               │
  │                              │  POST /tickets            │
  │                              │──────────────────►        │
  │                              │                           │
  │                              │  3. 返回 approval_url     │
  │                              │  ◄─────────────────       │
  │                              │                           │
  │                              │  4. 通知审批人             │
  │                              │───────────────────────────►│
  │                              │                           │
  │                              │                           │  5. 审批
  │                              │                           │  POST /tickets/:id/approve
  │                              │                           │──────────►
  │                              │                           │
  │                              │  6. 轮询结果               │
  │                              │  GET /tickets/:id         │
  │                              │──────────────────►        │
  │                              │  ◄─────────────────       │
```

---

## API 详细说明

### 公共接口

#### 登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"root","password":"your-password"}'
```

响应：

```json
{
  "token": "eyJhbG...",
  "role": "admin",
  "user_id": 1
}
```

---

### AI Agent 接口（API Key 认证）

#### 提交工单

```bash
curl -X POST http://localhost:8080/api/v1/tickets \
  -H "X-API-Key: lg_xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"project":"my-project","name":"deploy","args":{"env":"prod","region":"us-east-1"}}'
```

响应（dry-run 通过）：

```json
{
  "ticket_id": 1,
  "status": "pending_approval",
  "approval_url": "http://localhost:8080/tickets/1",
  "next_action": "任务已提交，需人工审批。请通知用户访问审批链接 http://localhost:8080/tickets/1 找审批人审批。审批通过后任务将自动执行，你可轮询 GET /api/v1/tickets/1 获取结果。"
}
```

响应（dry-run 失败）：

```json
{
  "ticket_id": 2,
  "status": "dryrun_failed",
  "approval_url": "http://localhost:8080/tickets/2",
  "next_action": "dry-run 校验未通过，任务未进入审批。请检查程序是否正确实现 --only-print（需输出 DRYRUN-OK 且退出码为 0）。详情见 dryrun_output。",
  "dryrun_output": "no marker\n---\n校验失败：dry-run 输出缺少 DRYRUN-OK 标记"
}
```

#### 轮询工单状态

```bash
curl http://localhost:8080/api/v1/tickets/1 \
  -H "X-API-Key: lg_xxxxxxxx"
```

响应（执行成功）：

```json
{
  "id": 1,
  "program_id": 1,
  "status": "done",
  "args": {
    "env": "prod"
  },
  "dryrun_output": "DRYRUN-OK\nwill deploy to prod",
  "exec_output": "Deploying to prod...",
  "approved_by": 2,
  "approved_at": "2026-05-29T10:30:00Z",
  "created_at": "2026-05-29T10:29:50Z"
}
```

响应（执行失败）：

```json
{
  "id": 2,
  "program_id": 1,
  "status": "exec_failed",
  "exec_output": "Error: connection refused",
  "created_at": "2026-05-29T10:29:50Z"
}
```

`status` 字段含义：

| status | 含义 |
|---|---|
| `executing` | 正在执行中，继续轮询 |
| `done` | 执行成功 |
| `exec_failed` | 执行失败 |

---

### 人工审批接口（JWT 认证）

#### 查看我的待审批列表

```bash
curl "http://localhost:8080/api/v1/tickets?status=pending_approval" \
  -H "Authorization: Bearer <token>"
```

#### 审批通过

```bash
curl -X POST http://localhost:8080/api/v1/tickets/1/approve \
  -H "Authorization: Bearer <token>"
```

审批通过后系统自动去掉 `--only-print` 参数执行真实命令。

#### 驳回

```bash
curl -X POST http://localhost:8080/api/v1/tickets/1/reject \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"reason":"风险过高，需要进一步评估"}'
```

---

### 管理接口（JWT + admin 角色）

#### 注册程序

```bash
curl -X POST http://localhost:8080/api/v1/programs \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "project": "my-project",
    "name": "deploy",
    "binary_path": "/usr/local/bin/my-deploy-tool",
    "approver_id": 2,
    "timeout_sec": 300,
    "params_schema": {
      "env": "string",
      "region": "string"
    }
  }'
```

注册时系统自动执行 `binary_path --help` 探测程序是否支持 `--only-print` 参数。不支持的程序将被拒绝注册。

#### 查看程序列表

```bash
curl http://localhost:8080/api/v1/programs \
  -H "Authorization: Bearer <admin-token>"
```

#### 更新程序配置

```bash
curl -X PUT http://localhost:8080/api/v1/programs/1 \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"enabled":false,"timeout_sec":600}'
```

#### 创建用户

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secure-pw","role":"user"}'
```

#### 创建 API Key

```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"hermes-agent"}'
```

响应（明文只返回这一次）：

```json
{
  "id": 1,
  "name": "hermes-agent",
  "api_key": "lg_a1b2c3d4e5f6..."
}
```

---

## 工单状态流转

```
pending_dryrun ──► pending_approval ──► approved ──► executing ──► done
       │                                        │
       ▼                                        ▼
 dryrun_failed                              exec_failed

pending_approval ──► rejected
```

| 状态                 | 含义                                     |
|--------------------|----------------------------------------|
| `pending_dryrun`   | 刚提交，等待 dry-run（内部状态，通常瞬间完成）            |
| `dryrun_failed`    | dry-run 校验失败（退出码非 0 或输出不含 `DRYRUN-OK`） |
| `pending_approval` | dry-run 通过，等待审批人操作                     |
| `approved`         | 审批人已批准（内部过渡状态，立即进入执行）                  |
| `executing`        | 正在执行真实命令                               |
| `done`             | 执行成功                                   |
| `exec_failed`      | 执行失败                                   |
| `rejected`         | 审批人已驳回                                 |

---

## 被托管程序的要求

LoopGuard 托管的命令行程序需要满足以下条件：

1. **支持 `--only-print` 参数**：传入此参数时，程序不执行真实操作，只打印将要执行的操作内容
2. **dry-run 输出包含 `DRYRUN-OK` 标记**：stdout 中必须包含字符串 `DRYRUN-OK`
3. **dry-run 退出码为 0**：正常退出表示参数合法
4. **支持 `--help` 参数**：注册时会自动探测帮助信息

示例（合规的部署脚本）：

```bash
#!/bin/bash
# /usr/local/bin/my-deploy

if [[ "$1" == "--help" ]]; then
  echo "Usage: my-deploy [--only-print] --env ENV"
  exit 0
fi

# 解析参数
ENV=""
ONLY_PRINT=false
while [[ $# -gt 0 ]]; do
  case $1 in
    --env) ENV="$2"; shift 2 ;;
    --only-print) ONLY_PRINT=true; shift ;;
    *) shift ;;
  esac
done

if $ONLY_PRINT; then
  echo "DRYRUN-OK"
  echo "Will deploy to environment: $ENV"
  exit 0
fi

# 真实部署逻辑
echo "Deploying to $ENV..."
```

注册此程序：

```bash
curl -X POST http://localhost:8080/api/v1/programs \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "project": "my-project",
    "name": "deploy",
    "binary_path": "/usr/local/bin/my-deploy",
    "approver_id": 2,
    "params_schema": {"env": "string"}
  }'
```

AI Agent 提交工单：

```bash
curl -X POST http://localhost:8080/api/v1/tickets \
  -H "X-API-Key: lg_eb77d3bd1fdec99d36c607a9b5fa8f6f0e10ba50ebef878b" \
  -H "Content-Type: application/json" \
  -d '{"project":"测试项目","name":"a+b","args":{"a":3, "b":4}}'
  
curl -X POST http://localhost:8080/api/v1/tickets \
  -H "X-API-Key: lg_775e92a40200e1a06c4cb6eb089289859981c0a153822236" \
  -H "Content-Type: application/json" \
  -d '{"project":"tsunami_ipban","name":"entry_ipban.py","args":{"-m": "group_unban", "-e": "never", "-i":"9.8.7.6", "-u":"huayang", "-g":"huayangblacklist", "--skip-confirm": ""}}'
```

审批人放行后，系统执行 `/usr/local/bin/my-deploy --env prod`（不带 `--only-print`）。

---

## 安全注意事项

- **API Key**：创建后明文只显示一次，丢失需重新创建
- **JWT Secret**：生产环境务必通过环境变量设置强随机密钥
- **审批人绑定**：每个程序注册时指定审批人，只有该审批人（或 admin）能放行
- **参数白名单**：`args` 的 key 必须在注册时的 `params_schema` 内，防止注入额外参数
- **保留字**：`only-print` 为系统保留，AI Agent 无法通过此参数绕过 dry-run
- **进程隔离**：执行命令时使用独立进程组，超时自动 kill 整组防残留子进程
