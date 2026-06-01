## ADDED Requirements

### Requirement: Program 的 interpreter 字段

Program 模型 SHALL 包含 `interpreter` 字段（string 类型，最大 256 字符）。该字段为可选字段，为空字符串时表示直接执行 `binary_path`，非空时表示使用该解释器执行 `binary_path`。

#### Scenario: interpreter 为空（Go 二进制）

- **WHEN** Program 的 `interpreter` 为空字符串
- **THEN** executor 直接执行 `binary_path`，命令为 `binary_path args...`

#### Scenario: interpreter 为 python3

- **WHEN** Program 的 `interpreter` 为 `"python3"`
- **THEN** executor 以 `python3 binary_path args...` 方式执行

#### Scenario: interpreter 为绝对路径

- **WHEN** Program 的 `interpreter` 为 `"/home/x/.venv/bin/python3"`
- **THEN** executor 以 `/home/x/.venv/bin/python3 binary_path args...` 方式执行

### Requirement: ExecRequest 传递 interpreter

`ExecRequest` 结构体 SHALL 包含 `Interpreter` 字段。`ProcessExecutor.Run()` SHALL 根据 `Interpreter` 是否为空选择执行方式。

#### Scenario: 带 interpreter 执行

- **WHEN** `ExecRequest.Interpreter` 为 `"python3"`，`BinaryPath` 为 `"/app/deploy.py"`，`Args` 为 `["--env", "prod"]`，`DryRun` 为 true
- **THEN** 执行命令为 `python3 /app/deploy.py --env prod --only-print`

#### Scenario: 不带 interpreter 执行

- **WHEN** `ExecRequest.Interpreter` 为空，`BinaryPath` 为 `"/usr/local/bin/tool"`，`Args` 为 `["--env", "prod"]`，`DryRun` 为 true
- **THEN** 执行命令为 `/usr/local/bin/tool --env prod --only-print`

### Requirement: Execution 记录完整命令

`ExecResult.Command` SHALL 记录实际执行的完整命令字符串，包含 interpreter（如有）。

#### Scenario: 带 interpreter 的命令记录

- **WHEN** interpreter 为 `"python3"`，binary_path 为 `"/app/deploy.py"`，args 为 `["--env", "prod"]`
- **THEN** `ExecResult.Command` 为 `"python3 /app/deploy.py --env prod"`

#### Scenario: 不带 interpreter 的命令记录

- **WHEN** interpreter 为空，binary_path 为 `"/usr/local/bin/tool"`，args 为 `["--env", "prod"]`
- **THEN** `ExecResult.Command` 为 `"/usr/local/bin/tool --env prod"`

### Requirement: 程序注册传递 interpreter

`ProgramService.Register()` SHALL 将 `RegisterInput.Interpreter` 传递给 executor 的 `--help` 探测请求，并存储到 Program 记录中。

#### Scenario: Python 脚本注册

- **WHEN** admin 注册程序时 `interpreter` 为 `"python3"`，`binary_path` 为 `"/app/deploy.py"`
- **THEN** 系统以 `python3 /app/deploy.py --help` 探测帮助信息，Program 记录的 `interpreter` 为 `"python3"`

### Requirement: 工单提交流程传递 interpreter

`TicketService.Submit()` 和 `TicketService.Approve()` SHALL 从 Program 读取 `Interpreter` 并传递给 `ExecRequest`。

#### Scenario: Python 脚本 dry-run

- **WHEN** AI 提交工单，对应 Program 的 `interpreter` 为 `"python3"`
- **THEN** dry-run 执行命令为 `python3 binary_path --param val --only-print`

#### Scenario: Python 脚本真实执行

- **WHEN** 审批人批准工单，对应 Program 的 `interpreter` 为 `"python3"`
- **THEN** 真实执行命令为 `python3 binary_path --param val`（不带 `--only-print`）
