## Context

LoopGuard 的 `ProcessExecutor` 使用 `exec.CommandContext(ctx, binaryPath, args...)` 执行被托管程序。这假定 `binaryPath` 是可直接执行的目标文件（Go 编译产物），但 Python 脚本、Shell 脚本等解释型语言需要通过解释器启动（如 `python3 script.py`）。

当前 Program 模型只有 `binary_path` 字段，没有区分"解释器"和"脚本路径"的概念。

## Goals / Non-Goals

**Goals:**
- Program 支持配置 `interpreter`，使解释型脚本（Python、Bash、Node 等）能被注册和执行
- 为空时完全兼容现有 Go 二进制执行方式
- 全链路（注册探测、dry-run、真实执行）正确使用 interpreter
- 前端管理页面支持配置和展示 interpreter

**Non-Goals:**
- 不自动检测文件类型或推荐解释器
- 不管理解释器环境（虚拟环境、版本切换等）
- 不改变 `--only-print` 协议本身
- 不支持多命令拼接（如 `bash -c "..."` 复杂命令）

## Decisions

### D1: 新增 `interpreter` 字段而非扩展 `binary_path`

**选择**: Program 新增独立的 `interpreter string` 字段

**备选方案**:
- A. `binary_path` 含空格拆分（如 `"python3 /path/to/script.py"`）→ hacky，参数解析歧义
- B. 按扩展名自动检测（`.py` → python3）→ 不够灵活，无法处理虚拟环境路径

**理由**: 独立字段语义清晰，空值 = 直接执行（Go 二进制），非空 = 解释器路径。支持任意解释器绝对路径（如 `/home/x/.venv/bin/python3`），不做假设。

### D2: Executor 分支逻辑

**选择**: `ProcessExecutor.Run()` 内部判断 `req.Interpreter != ""`

```go
if req.Interpreter != "" {
    cmd = exec.CommandContext(runCtx, req.Interpreter, append([]string{req.BinaryPath}, args...)...)
} else {
    cmd = exec.CommandContext(runCtx, req.BinaryPath, args...)
}
```

**理由**: 改动最小，只改 executor 内部。调用方（service 层）只需传递 interpreter，不需要知道执行细节。

### D3: interpreter 字段可选且无服务端校验

**选择**: 不校验 interpreter 路径是否存在

**理由**: 注册时 `--help` 探测本身就是验证——如果 interpreter 不存在，`exec.CommandContext` 会返回错误，注册直接失败。无需额外校验逻辑。

### D4: UpdateProgram 支持修改 interpreter

**选择**: 编辑表单包含 interpreter 字段

**理由**: 用户可能初始注册时漏填或填错 interpreter，应支持修改而非只能删除重建。

## Risks / Trade-offs

- **[interpreter 路径安全]** → interpreter 理论上可以是任意命令（如 `rm`）。目前 LoopGuard 部署模型是内部信任的（admin 注册程序），暂不做路径白名单限制。
- **[GORM AutoMigrate 兼容性]** → 新增 nullable 列，默认空字符串，对已有数据完全兼容。已验证 GORM AutoMigrate 支持 `default:''` 的新列。
- **[信号传递]** → 现有 `Setpgid: true` + `SIGKILL` 进程组 kill 机制不受影响——interpreter 进程和脚本在同一进程组内，kill 整组仍有效。
