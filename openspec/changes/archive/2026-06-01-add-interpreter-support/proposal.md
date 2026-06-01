## Why

当前 LoopGuard 只支持直接执行二进制文件（Go 编译产物），`ProcessExecutor` 将 `BinaryPath` 作为命令名直接传给 `exec.CommandContext`。Python 脚本、Shell 脚本等解释型语言需要通过解释器执行（如 `python3 script.py`），无法被注册和运行。

## What Changes

- Program 模型新增 `interpreter` 字段（可选），为空时保持现有直接执行行为，有值时以 `interpreter binary_path args...` 方式执行
- `ProcessExecutor.Run()` 支持通过 interpreter 调度执行
- 程序注册（Register）、工单提交（Submit）、审批执行（Approve）全链路传递 interpreter
- 管理页面支持配置 interpreter

## Capabilities

### New Capabilities

- `interpreter-execution`: Program 通过解释器执行的能力，覆盖 executor 分支逻辑、命令字符串构建、全链路传递

### Modified Capabilities

- `admin-panel`: 程序管理表单和列表新增 interpreter 字段的展示与编辑
- `loopguard-script-protocol`: 被托管程序不再限定为独立二进制，支持通过解释器间接执行

## Impact

- **Model**: `Program` 新增 `Interpreter` 列（GORM AutoMigrate，向后兼容）
- **Executor**: `ExecRequest` 新增字段，`ProcessExecutor.Run()` 新增分支逻辑
- **Service**: `RegisterInput`、`Submit()`、`Approve()` 传递 interpreter
- **API**: `CreateProgram` / `UpdateProgram` 接收 interpreter 参数
- **Frontend**: 程序表单和类型定义新增 interpreter
- **Tests**: executor 和 service 测试覆盖 interpreter 分支
