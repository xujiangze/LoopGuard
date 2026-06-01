## 1. Model 层

- [x] 1.1 `internal/model/models.go` — Program 结构体新增 `Interpreter string` 字段（`gorm:"size:256;default:''" json:"interpreter"`）

## 2. Executor 层

- [x] 2.1 `internal/executor/executor.go` — ExecRequest 新增 `Interpreter string` 字段
- [x] 2.2 `internal/executor/process.go` — Run() 根据 Interpreter 是否为空选择执行方式，buildCommandString() 记录完整命令
- [x] 2.3 `internal/executor/process_test.go` — 新增测试：interpreter 为空 / 非空时的命令构建和执行行为

## 3. Service 层

- [x] 3.1 `internal/service/program.go` — RegisterInput 新增 `Interpreter` 字段；Register() 传递到 ExecRequest 并写入 Program
- [x] 3.2 `internal/service/ticket.go` — Submit() 从 Program 读取 Interpreter 传给 ExecRequest
- [x] 3.3 `internal/service/ticket.go` — Approve() 从 Program 读取 Interpreter 传给 ExecRequest
- [x] 3.4 `internal/service/ticket_test.go` — 更新 seedProgram 和现有测试用例覆盖 Interpreter

## 4. API 层

- [x] 4.1 `internal/api/admin_handler.go` — CreateProgram 请求体新增 `interpreter` 字段，传给 RegisterInput
- [x] 4.2 `internal/api/admin_handler.go` — UpdateProgram 请求体新增 `interpreter` 字段（指针类型，可选更新）

## 5. 前端

- [x] 5.1 `web/src/types/index.ts` — Program 接口新增 `interpreter: string`
- [x] 5.2 `web/src/pages/ProgramPage.tsx` — 创建表单新增 interpreter 输入框（可选，placeholder 提示如 "python3"）
- [x] 5.3 `web/src/pages/ProgramPage.tsx` — 编辑表单新增 interpreter 字段
- [x] 5.4 `web/src/pages/ProgramPage.tsx` — 程序列表路径列：interpreter 非空时显示 `interpreter binary_path`

## 6. 验证

- [x] 6.1 运行 `go test ./...` 确认全部测试通过
- [ ] 6.2 启动服务，通过管理页面注册一个 Python 脚本程序，验证全链路（注册 → dry-run → 审批 → 执行）
