## MODIFIED Requirements

### Requirement: --only-print 参数支持

脚本 SHALL 接受 `--only-print` 命令行参数。当此参数存在时，脚本 SHALL 进入 dry-run 模式，不执行任何写操作。LoopGuard 平台 SHALL 支持通过解释器（interpreter）间接执行脚本时正确传递 `--only-print` 参数。

#### Scenario: 带 --only-print 执行

- **WHEN** 脚本以 `--only-print` 参数启动
- **THEN** 脚本不执行任何写操作（HTTP 写请求、DB 写操作、文件写入、Shell 命令），只打印将要执行的请求内容，退出码为 0

#### Scenario: 不带 --only-print 执行

- **WHEN** 脚本不带 `--only-print` 参数启动
- **THEN** 脚本正常执行所有操作，行为与改造前一致

#### Scenario: 通过解释器执行 Python 脚本的 dry-run

- **WHEN** Program 配置 `interpreter` 为 `"python3"`，AI 提交工单触发 dry-run
- **THEN** 平台执行 `python3 /path/to/script.py --param val --only-print`，脚本的 `--only-print` 参数正确传递到脚本进程

#### Scenario: 通过解释器执行 Python 脚本的真实执行

- **WHEN** Program 配置 `interpreter` 为 `"python3"`，审批通过后真实执行
- **THEN** 平台执行 `python3 /path/to/script.py --param val`，不带 `--only-print`
