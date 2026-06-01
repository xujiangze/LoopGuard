# 程序注册改造：文件上传 + 去白名单

> 日期：2026-06-01

## 背景

当前程序注册采用填写服务器本地 `binary_path` + `params_schema` 参数白名单的方式。实际使用中：
- 程序文件散落在服务器各处，不在平台管理
- 参数白名单对 AI agent 来说没有实际价值，AI 只需要看 `--help` 输出即可理解用法
- 工单提交时 args 是 `map[string]any`，强制 `--key value` 形式，不够灵活

## 目标

1. 程序通过**上传文件**注册，平台统一管理程序文件
2. 引入 `project/name` 层级组织文件
3. 去掉参数白名单，用 `--help` 输出作为程序说明
4. 工单提交 args 改为 `[]string`（shell 风格参数列表）
5. 更新程序时自动重跑 `--help` 刷新说明
6. 基本的注入防御

## 设计决策

- **文件存储**：本地文件系统 `{WORKSPACE_DIR}/{project}/{name}/`，全量覆盖
- **上传方式**：单次 multipart/form-data 上传全部文件
- **解释器**：注册时明确指定（python3、bash 等）
- **项目层级**：project 保持字符串分组字段，不做独立 CRUD
- **注入防御**：检查元字符 `[;&|`$(){}><!]` + `--only-print` 保留字

## 数据模型变更

### Program 模型

```go
type Program struct {
    ID             uint64     `gorm:"primaryKey"`
    Project        string     `gorm:"size:128;not null;uniqueIndex:uk_project_name"`
    Name           string     `gorm:"size:128;not null;uniqueIndex:uk_project_name"`
    EntryFile      string     `gorm:"size:256;not null" json:"entry_file"`
    Interpreter    string     `gorm:"size:256;default:''"`
    HelpText       string     `gorm:"type:text"`
    ApproverID     uint64     `gorm:"not null"`
    TimeoutSec     int        `gorm:"not null;default:300"`
    SupportsDryrun bool       `gorm:"not null;default:true"`
    Enabled        bool       `gorm:"not null;default:true"`
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

变更：
- `BinaryPath` → `EntryFile`：存入口文件名（如 `entry_ipban.py`）
- 删除 `ParamsSchema` 字段
- 新增 `UpdatedAt` 字段

### Ticket 模型

Args 字段存储格式从 `map[string]any` 改为 `[]string`：
```json
["-m", "group_unban", "-e", "never", "--skip-confirm"]
```

### 命名约束

project 和 name 只允许 `[a-zA-Z0-9_-]`，用于构建文件系统路径时防止注入。

### 文件存储结构

```
{WORKSPACE_DIR}/
  tsunami_ipban/           # project
    entry_ipban/           # name (程序名)
      entry_ipban.py       # 入口文件
      utils.py             # 辅助文件
      config.json          # 辅助文件
```

## API 接口变更

### 注册程序（替换原 POST /api/v1/programs）

```
POST /api/v1/programs
Content-Type: multipart/form-data
认证: JWT + admin

字段：
  project      string   必填
  name         string   必填
  entry_file   string   必填，入口文件名（必须在 files 中）
  interpreter  string   必填
  approver_id  int      必填
  timeout_sec  int      可选，默认 300
  files        []File   必填，至少包含入口文件
```

流程：保存文件 → 执行 `{interpreter} {entry_file} --help` → 校验 `--only-print` 支持 → 存库。

### 更新程序（替换原 PUT /api/v1/programs/:id）

```
PUT /api/v1/programs/:id
Content-Type: multipart/form-data
认证: JWT + admin

字段：
  entry_file   string   可选
  interpreter  string   可选
  approver_id  int      可选
  timeout_sec  int      可选
  enabled      bool     可选
  files        []File   可选，传了则全量覆盖目录
```

流程：有 files 则覆盖目录 → 重新执行 `--help` → 更新库记录。

### 工单提交（改动 POST /api/v1/tickets）

```json
{
  "project": "tsunami_ipban",
  "name": "entry_ipban",
  "args": ["-m", "group_unban", "-e", "never", "-i", "9.8.7.6", "--skip-confirm"]
}
```

args 从 `map[string]any` 改为 `[]string`。

### AI 获取程序列表

```
GET /api/v1/programs
认证: API Key 或 JWT
```

新增对 API Key 认证的支持，AI agent 可获取程序列表（含 HelpText）了解用法。

## 注入防御

### args 元字符检查

```go
var dangerousPatterns = regexp.MustCompile(`[;&|` + "`" + `$(){}><!]`)

func validateArgs(args []string) error {
    for _, arg := range args {
        if arg == "--only-print" {
            return errors.New("--only-print 为系统保留参数，禁止传入")
        }
        if dangerousPatterns.MatchString(arg) {
            return fmt.Errorf("参数包含危险字符: %s", arg)
        }
    }
    return nil
}
```

### 上传文件防御

- 文件名 sanitize：禁止 `/`、`..`、空字符，防路径穿越
- 单文件大小上限：10MB
- 单次上传文件数上限：20 个
- 上传文件 `chmod -x`，由 interpreter 负责执行
- 入口文件名必须在上传文件列表中存在

### 执行安全（保持现有机制）

- 独立进程组 + 超时 kill 整组
- WorkDir 设为 `workspace/{project}/{name}/`

## 执行流程变更

### 命令构建

args 直接透传 `[]string`，不再从 map 拼装：

```go
binaryPath := filepath.Join(workspaceDir, p.Project, p.Name, p.EntryFile)
args := append(ticketArgs, "--only-print")  // dry-run 时追加

if p.Interpreter != "" {
    cmd = exec.CommandContext(ctx, p.Interpreter, append([]string{binaryPath}, args...)...)
} else {
    cmd = exec.CommandContext(ctx, binaryPath, args...)
}
```

### 完整流程

1. AI 调 `GET /api/v1/programs` 获取程序列表和 HelpText
2. AI 按 HelpText 说明构造 args 列表，调 `POST /api/v1/tickets`
3. validateArgs（元字符 + 保留字）→ 通过
4. 执行 dry-run：`{interpreter} {workspace}/{project}/{name}/{entry_file} {args...} --only-print`
5. 检查 DRYRUN-OK → 进入审批
6. 审批通过 → 去掉 `--only-print` 真实执行

## 前端改动

- 程序管理页：表单改为文件上传（支持多文件 + 指定入口文件名）
- 去掉参数白名单编辑区域
- 程序列表/详情展示 HelpText 代替 params_schema

## 删除的代码

- `validateArgs` 中的白名单逻辑（替换为元字符检查版本）
- `buildArgs` 函数（不再需要从 map 拼装）
- `ParamsSchema` 相关的所有字段和逻辑
- `RegisterInput.ParamsSchema` 字段
- admin handler 中 CreateProgram/UpdateProgram 的 JSON 绑定（改为 multipart）

## 迁移策略

- GORM AutoMigrate 自动处理字段增减（新增 `entry_file`、`updated_at`，删除 `params_schema`）
- 现有数据不兼容（binary_path 语义变化），需清空 programs 表或手动迁移
