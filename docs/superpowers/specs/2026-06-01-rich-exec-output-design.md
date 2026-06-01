# 设计文档：丰富工单执行输出

> 日期: 2026-06-01
> 状态: 已批准

## 背景

AI agent 通过 `POST /api/v1/tickets` 提交工单后，当前存在两个问题：

1. **执行命令不可见**：AI 无法看到后端实际拼装执行的命令（如 `python3 /path/script.py --target prod --only-print`），缺乏调试依据。
2. **Dry-run 输出不完整**：`dryrun_output` 只保存了 stdout，丢弃了 stderr。脚本报错时（如 import 失败、语法错误），AI 拿不到错误信息，无法定位问题。

真实执行（审批通过后）同样存在上述问题。

## 方案

### 核心思路

将 `dryrun_output` 和新增的 `exec_output` 从"原始 stdout 字符串"升级为"Markdown 格式的完整执行报告"，包含执行命令、stdout、stderr、退出码和校验结果。

零 schema 破坏性变更，GORM AutoMigrate 自动处理新字段。

### 输出格式

使用 Markdown `#` 标题分层：

**成功示例（dry-run）**：
```markdown
# 命令
python3 /path/to/script.py --target prod --only-print

# stdout
将执行: kubectl apply -f deployment.yaml --namespace prod
目标环境: production
DRYRUN-OK

# stderr
(无)

# 结果
退出码: 0 | 校验: 通过
```

**失败示例**：
```markdown
# 命令
python3 /path/to/script.py --target prod --only-print

# stdout

# stderr
Traceback (most recent call last):
  File "/path/to/script.py", line 3, in <module>
    import missing_module
ModuleNotFoundError: No module named 'missing_module'

# 结果
退出码: 1 | 校验: 失败 - dry-run 退出码非 0（实际 1）
```

**真实执行报告**：
```markdown
# 命令
python3 /path/to/script.py --target prod

# stdout
deployment.apps/my-app configured

# stderr
(无)

# 结果
退出码: 0 | 耗时: 1523ms
```

### 格式规则

- `# 命令`：完整的执行命令字符串（含 interpreter + binary + args + `--only-print`）
- `# stdout`：原样输出脚本的标准输出，为空时留空
- `# stderr`：原样输出脚本的标准错误，为空时输出 `(无)`
- `# 结果`：
  - dry-run：`退出码: X | 校验: 通过/失败原因`
  - 真实执行：`退出码: X | 耗时: Xms`

## 数据模型变更

### Ticket 模型新增字段

```go
// internal/model/models.go
type Ticket struct {
    // ... 现有字段 ...
    DryrunOutput string `gorm:"type:mediumtext" json:"dryrun_output"` // 内容从纯 stdout 升级为完整报告
    ExecOutput   string `gorm:"type:mediumtext" json:"exec_output"`   // 新增：真实执行报告
    // ...
}
```

GORM AutoMigrate 自动添加 `exec_output` 列，无需手动迁移。

## 代码改动

### 1. 新增格式化函数

文件：`internal/service/ticket.go`

```go
func formatExecReport(command, stdout, stderr string, exitCode int, result string) string {
    var sb strings.Builder
    sb.WriteString("# 命令\n")
    sb.WriteString(command)
    sb.WriteString("\n\n# stdout\n")
    sb.WriteString(stdout)
    sb.WriteString("\n\n# stderr\n")
    if stderr == "" {
        sb.WriteString("(无)")
    } else {
        sb.WriteString(stderr)
    }
    sb.WriteString("\n\n# 结果\n")
    sb.WriteString(result)
    return sb.String()
}
```

### 2. Submit 方法改动

文件：`internal/service/ticket.go`，`Submit` 方法

替换所有 `tk.DryrunOutput` 赋值为 `formatExecReport` 调用：

| 原代码 | 新代码 |
|---|---|
| `tk.DryrunOutput = "dry-run 执行错误：" + errString(runErr)` | 需从 `res.Command` 取值（res 可能为 nil 时使用 `"N/A"`）：`tk.DryrunOutput = formatExecReport(cmd, "", "", -1, "执行错误: "+errString(runErr))` |
| `tk.DryrunOutput = res.Stdout` (通过路径) | `tk.DryrunOutput = formatExecReport(res.Command, res.Stdout, res.Stderr, res.ExitCode, "校验: 通过")` |
| `res.Stdout + "\n---\n校验失败：" + v.Reason` (失败路径) | `formatExecReport(res.Command, res.Stdout, res.Stderr, res.ExitCode, "校验: 失败 - "+v.Reason)` |

### 3. Approve 方法改动

文件：`internal/service/ticket.go`，`Approve` 方法

真实执行后，新增 `tk.ExecOutput` 赋值：

```go
tk.ExecOutput = formatExecReport(
    res.Command, res.Stdout, res.Stderr, res.ExitCode,
    fmt.Sprintf("耗时: %dms", res.DurationMs),
)
```

### 4. AI Handler 改动

文件：`internal/api/ai_handler.go`，`Submit` handler

`pending_approval` 状态的响应也返回 `dryrun_output`（当前只有 `dryrun_failed` 才返回）：

```go
case model.StatusPendingApproval:
    resp["next_action"] = "..."
    resp["dryrun_output"] = tk.DryrunOutput  // 新增
```

## API 响应变更

### POST /api/v1/tickets

成功进入审批时新增 `dryrun_output` 字段：
```json
{
  "ticket_id": 1,
  "status": "pending_approval",
  "approval_url": "http://localhost:8080/tickets/1",
  "next_action": "...",
  "dryrun_output": "# 命令\n..."
}
```

### GET /api/v1/tickets/:id

Ticket JSON 自然包含 `dryrun_output`（已有）和 `exec_output`（新增）：
```json
{
  "id": 1,
  "status": "done",
  "dryrun_output": "# 命令\n...",
  "exec_output": "# 命令\n...",
  ...
}
```

## 兼容性

- 旧数据 `exec_output` 为空字符串，前端显示为空即可
- 旧数据 `dryrun_output` 为纯 stdout（无 `# 命令` 标记），前端/AI 需容错
- 无破坏性 API 变更，所有新增字段为追加
