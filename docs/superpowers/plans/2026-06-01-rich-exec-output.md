# 丰富工单执行输出 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 dry-run 和真实执行的输出从纯 stdout 升级为 Markdown 格式的完整执行报告（含命令、stdout、stderr、退出码、校验结果）。

**Architecture:** 在 service 层新增 `formatExecReport` 函数统一格式化执行报告。Ticket 模型新增 `ExecOutput` 字段存储真实执行报告。`dryrun_output` 的赋值从直接取 `res.Stdout` 改为调用格式化函数。AI handler 提交响应在 `pending_approval` 状态也返回 `dryrun_output`。

**Tech Stack:** Go, GORM, Gin, testify

---

### Task 1: 新增 `formatExecReport` 函数及测试

**Files:**
- Modify: `internal/service/ticket.go` (新增函数)
- Test: `internal/service/ticket_test.go` (新增测试)

- [ ] **Step 1: 为 `formatExecReport` 写测试**

在 `internal/service/ticket_test.go` 末尾添加：

```go
func TestFormatExecReport(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		stdout   string
		stderr   string
		exitCode int
		result   string
		want     []string // 必须包含的子串
	}{
		{
			name: "dryrun pass with output",
			command: "python3 /bin/deploy --env prod --only-print",
			stdout:  "将执行: kubectl apply\nDRYRUN-OK",
			stderr:  "",
			exitCode: 0,
			result:  "校验: 通过",
			want: []string{
				"# 命令\npython3 /bin/deploy --env prod --only-print",
				"# stdout\n将执行: kubectl apply\nDRYRUN-OK",
				"# stderr\n(无)",
				"# 结果\n退出码: 0 | 校验: 通过",
			},
		},
		{
			name: "exec fail with stderr",
			command:  "/bin/deploy --env prod",
			stdout:   "",
			stderr:   "error: connection refused",
			exitCode: 1,
			result:   "校验: 失败 - dry-run 退出码非 0（实际 1）",
			want: []string{
				"# 命令\n/bin/deploy --env prod",
				"# stdout\n",
				"# stderr\nerror: connection refused",
				"# 结果\n退出码: 1 | 校验: 失败 - dry-run 退出码非 0（实际 1）",
			},
		},
		{
			name: "real exec with duration",
			command:  "python3 /bin/deploy --env prod",
			stdout:   "deployment configured",
			stderr:   "",
			exitCode: 0,
			result:   "耗时: 1523ms",
			want: []string{
				"# 命令\npython3 /bin/deploy --env prod",
				"# stdout\ndeployment configured",
				"# stderr\n(无)",
				"# 结果\n退出码: 0 | 耗时: 1523ms",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatExecReport(tt.command, tt.stdout, tt.stderr, tt.exitCode, tt.result)
			for _, w := range tt.want {
				assert.Contains(t, got, w)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/service/ -run TestFormatExecReport -v`
Expected: FAIL — `formatExecReport` undefined

- [ ] **Step 3: 实现 `formatExecReport`**

在 `internal/service/ticket.go` 的 `errString` 函数后面添加：

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
	sb.WriteString(fmt.Sprintf("退出码: %d | %s", exitCode, result))
	return sb.String()
}
```

注意：`strings` 包已在 import 中（`ticket.go` 没有用到 strings，检查一下 import）。当前 `ticket.go` 没有导入 `strings`，需要添加。

在 `ticket.go` 的 import 块中添加 `"strings"`：

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	// ...
)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/service/ -run TestFormatExecReport -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/service/ticket.go internal/service/ticket_test.go
git commit -m "feat: add formatExecReport for Markdown execution reports"
```

---

### Task 2: Ticket 模型新增 `ExecOutput` 字段

**Files:**
- Modify: `internal/model/models.go:47-62` (Ticket struct)

- [ ] **Step 1: 在 Ticket struct 添加 ExecOutput 字段**

在 `internal/model/models.go` 的 `Ticket` struct 中，`DryrunOutput` 字段后面添加 `ExecOutput`：

```go
type Ticket struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	ProgramID    uint64         `gorm:"not null;index" json:"program_id"`
	Args         datatypes.JSON `gorm:"type:json;not null" json:"args"`
	Status       TicketStatus   `gorm:"type:varchar(32);not null;index" json:"status"`
	SubmittedBy  uint64         `gorm:"not null" json:"submitted_by"`
	ApproverID   uint64         `gorm:"not null;index" json:"approver_id"`
	DryrunOutput string         `gorm:"type:mediumtext" json:"dryrun_output"`
	ExecOutput   string         `gorm:"type:mediumtext" json:"exec_output"`
	ApprovedBy   *uint64        `json:"approved_by"`
	ApprovedAt   *time.Time     `json:"approved_at"`
	RejectReason string         `gorm:"size:512" json:"reject_reason"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: 验证现有测试仍然通过（GORM AutoMigrate 会处理新字段）**

Run: `go test ./...`
Expected: PASS（所有现有测试不受影响）

- [ ] **Step 3: 提交**

```bash
git add internal/model/models.go
git commit -m "feat: add ExecOutput field to Ticket model"
```

---

### Task 3: 改造 `Submit` 方法使用 `formatExecReport`

**Files:**
- Modify: `internal/service/ticket.go:74-88` (Submit 方法中 DryrunOutput 赋值)

- [ ] **Step 1: 更新 `TestSubmitTicketDryrunPass` 断言**

在 `internal/service/ticket_test.go` 中，将 `TestSubmitTicketDryrunPass` 的断言改为验证新格式：

```go
func TestSubmitTicketDryrunPass(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "/bin/deploy --env prod --only-print",
		ExitCode: 0, Stdout: "DRYRUN-OK\nwill deploy", Stderr: "",
	}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"env": "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "# 命令")
	assert.Contains(t, tk.DryrunOutput, "DRYRUN-OK")
	assert.Contains(t, tk.DryrunOutput, "校验: 通过")
}
```

- [ ] **Step 2: 更新 `TestSubmitTicketDryrunFail` 断言**

```go
func TestSubmitTicketDryrunFail(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "/bin/deploy --env prod --only-print",
		ExitCode: 0, Stdout: "no marker", Stderr: "",
	}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7, Args: map[string]any{"env": "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDryrunFailed, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "# 命令")
	assert.Contains(t, tk.DryrunOutput, "校验: 失败")
}
```

- [ ] **Step 3: 新增测试 — dry-run 执行错误（runErr != nil）**

在 `ticket_test.go` 末尾添加：

```go
func TestSubmitTicketDryrunRunError(t *testing.T) {
	fe := &fakeExecutor{result: nil, err: errors.New("binary not found")}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"env": "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDryrunFailed, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "# 命令")
	assert.Contains(t, tk.DryrunOutput, "binary not found")
}
```

注意：需要在 `ticket_test.go` 的 import 中添加 `"errors"`。

- [ ] **Step 4: 新增测试 — dry-run 有 stderr 输出**

```go
func TestSubmitTicketDryrunWithStderr(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "python3 /bin/deploy --env prod --only-print",
		ExitCode: 0, Stdout: "DRYRUN-OK", Stderr: "warning: deprecated API",
	}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"env": "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "# stderr\nwarning: deprecated API")
}
```

- [ ] **Step 5: 运行测试确认失败**

Run: `go test ./internal/service/ -run "TestSubmitTicket" -v`
Expected: 新断言 FAIL（DryrunOutput 还是旧格式）

- [ ] **Step 6: 改造 Submit 方法的 DryrunOutput 赋值**

替换 `internal/service/ticket.go` 的 `Submit` 方法中第 74-88 行：

**原代码：**
```go
	if runErr != nil || res == nil {
		tk.Status = model.StatusDryrunFailed
		tk.DryrunOutput = "dry-run 执行错误：" + errString(runErr)
		_ = svc.store.UpdateTicket(tk)
		return tk, nil
	}
	tk.DryrunOutput = res.Stdout
	if v := ValidateDryrun(res); !v.Passed {
		tk.Status = model.StatusDryrunFailed
		tk.DryrunOutput = res.Stdout + "\n---\n校验失败：" + v.Reason
	} else {
		tk.Status = model.StatusPendingApproval
	}
	_ = svc.store.UpdateTicket(tk)
	return tk, nil
```

**替换为：**
```go
	if runErr != nil || res == nil {
		cmd := "N/A"
		if res != nil {
			cmd = res.Command
		}
		tk.Status = model.StatusDryrunFailed
		tk.DryrunOutput = formatExecReport(cmd, "", "", -1, "执行错误: "+errString(runErr))
		_ = svc.store.UpdateTicket(tk)
		return tk, nil
	}

	v := ValidateDryrun(res)
	if !v.Passed {
		tk.Status = model.StatusDryrunFailed
		tk.DryrunOutput = formatExecReport(res.Command, res.Stdout, res.Stderr, res.ExitCode, "校验: 失败 - "+v.Reason)
	} else {
		tk.Status = model.StatusPendingApproval
		tk.DryrunOutput = formatExecReport(res.Command, res.Stdout, res.Stderr, res.ExitCode, "校验: 通过")
	}
	_ = svc.store.UpdateTicket(tk)
	return tk, nil
```

- [ ] **Step 7: 运行测试确认通过**

Run: `go test ./internal/service/ -run "TestSubmitTicket|TestFormatExecReport" -v`
Expected: PASS

- [ ] **Step 8: 运行全部测试确认无回归**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 9: 提交**

```bash
git add internal/service/ticket.go internal/service/ticket_test.go
git commit -m "feat: use formatExecReport for dryrun output in Submit"
```

---

### Task 4: 改造 `Approve` 方法填充 `ExecOutput`

**Files:**
- Modify: `internal/service/ticket.go:180-185` (Approve 方法)

- [ ] **Step 1: 新增测试 — 审批后 ExecOutput 包含执行报告**

在 `ticket_test.go` 末尾添加：

```go
func TestApproveFillsExecOutput(t *testing.T) {
	// 先创建一个 pending_approval 的工单
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "/bin/deploy --env prod", ExitCode: 0,
		Stdout: "deployment configured", Stderr: "", DurationMs: 1523,
	}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)

	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(`{"env":"prod"}`),
		Status: model.StatusPendingApproval, SubmittedBy: 7,
		ApproverID: p.ApproverID,
		DryrunOutput: "# 命令\n/bin/deploy --only-print\n\n# 结果\n退出码: 0 | 校验: 通过",
	}
	require.NoError(t, s.CreateTicket(tk))

	result, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusDone, result.Status)
	assert.Contains(t, result.ExecOutput, "# 命令")
	assert.Contains(t, result.ExecOutput, "deployment configured")
	assert.Contains(t, result.ExecOutput, "耗时: 1523ms")
}
```

需要在 `ticket_test.go` import 中添加 `"gorm.io/datatypes"`（如果还没有）。

- [ ] **Step 2: 新增测试 — 审批后执行失败**

```go
func TestApproveExecFailedFillsExecOutput(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "/bin/deploy --env prod", ExitCode: 1,
		Stdout: "", Stderr: "error: connection refused", DurationMs: 500,
	}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)

	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(`{"env":"prod"}`),
		Status: model.StatusPendingApproval, SubmittedBy: 7,
		ApproverID: p.ApproverID,
		DryrunOutput: "# 命令\n/bin/deploy --only-print",
	}
	require.NoError(t, s.CreateTicket(tk))

	result, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusExecFailed, result.Status)
	assert.Contains(t, result.ExecOutput, "# stderr\nerror: connection refused")
	assert.Contains(t, result.ExecOutput, "退出码: 1")
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/service/ -run "TestApprove" -v`
Expected: FAIL — ExecOutput 为空

- [ ] **Step 4: 改造 Approve 方法**

替换 `internal/service/ticket.go` 的 `Approve` 方法中第 180-185 行：

**原代码：**
```go
	if runErr != nil || res == nil || res.ExitCode != 0 || res.TimedOut {
		tk.Status = model.StatusExecFailed
	} else {
		tk.Status = model.StatusDone
	}
	_ = svc.store.UpdateTicket(tk)
	return tk, nil
```

**替换为：**
```go
	if runErr != nil || res == nil {
		cmd := "N/A"
		if res != nil {
			cmd = res.Command
		}
		tk.Status = model.StatusExecFailed
		tk.ExecOutput = formatExecReport(cmd, "", "", -1, "执行错误: "+errString(runErr))
	} else if res.ExitCode != 0 || res.TimedOut {
		tk.Status = model.StatusExecFailed
		tk.ExecOutput = formatExecReport(res.Command, res.Stdout, res.Stderr, res.ExitCode,
			fmt.Sprintf("耗时: %dms", res.DurationMs))
	} else {
		tk.Status = model.StatusDone
		tk.ExecOutput = formatExecReport(res.Command, res.Stdout, res.Stderr, res.ExitCode,
			fmt.Sprintf("耗时: %dms", res.DurationMs))
	}
	_ = svc.store.UpdateTicket(tk)
	return tk, nil
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/service/ -run "TestApprove" -v`
Expected: PASS

- [ ] **Step 6: 运行全部测试确认无回归**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add internal/service/ticket.go internal/service/ticket_test.go
git commit -m "feat: fill ExecOutput in Approve with formatted execution report"
```

---

### Task 5: AI Handler 提交响应返回 `dryrun_output`

**Files:**
- Modify: `internal/api/ai_handler.go:47-51` (Submit handler pending_approval 分支)
- Modify: `internal/api/ai_handler_test.go` (更新测试)

- [ ] **Step 1: 更新 `TestAISubmitReturnsApprovalURL` 验证 `dryrun_output`**

在 `internal/api/ai_handler_test.go` 中更新现有测试并添加新测试：

```go
func TestAISubmitReturnsApprovalURL(t *testing.T) {
	r, _ := setupAI(t)
	body, _ := json.Marshal(map[string]any{
		"project": "demo", "name": "deploy", "args": map[string]any{"msg": "DRYRUN-OK"}})
	req := httptest.NewRequest("POST", "/api/v1/tickets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["approval_url"])
	assert.NotEmpty(t, resp["next_action"])
	assert.NotZero(t, resp["ticket_id"])
	// 新增：pending_approval 状态也应返回 dryrun_output
	assert.NotEmpty(t, resp["dryrun_output"])
	dryrunOut, ok := resp["dryrun_output"].(string)
	require.True(t, ok)
	assert.Contains(t, dryrunOut, "# 命令")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/api/ -run TestAISubmitReturnsApprovalURL -v`
Expected: FAIL — `resp["dryrun_output"]` 为 nil

- [ ] **Step 3: 修改 ai_handler.go**

在 `internal/api/ai_handler.go` 的 `Submit` 方法中，`pending_approval` 分支添加 `dryrun_output`：

**原代码（第 48-51 行）：**
```go
		case model.StatusPendingApproval:
			resp["next_action"] = fmt.Sprintf(
				"任务已提交，需人工审批。请通知用户访问审批链接 %s 找审批人审批。审批通过后任务将自动执行，你可轮询 GET /api/v1/tickets/%d 获取结果。",
				url, tk.ID)
```

**替换为：**
```go
		case model.StatusPendingApproval:
			resp["next_action"] = fmt.Sprintf(
				"任务已提交，需人工审批。请通知用户访问审批链接 %s 找审批人审批。审批通过后任务将自动执行，你可轮询 GET /api/v1/tickets/%d 获取结果。",
				url, tk.ID)
			resp["dryrun_output"] = tk.DryrunOutput
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/api/ -run TestAISubmitReturnsApprovalURL -v`
Expected: PASS

- [ ] **Step 5: 运行全部测试确认无回归**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/api/ai_handler.go internal/api/ai_handler_test.go
git commit -m "feat: return dryrun_output in submit response for pending_approval"
```

---

### Task 6: 前端 TypeScript 类型更新

**Files:**
- Modify: `web/src/types/index.ts` (Ticket 接口新增 exec_output)

- [ ] **Step 1: 在 Ticket interface 添加 exec_output 字段**

在 `web/src/types/index.ts` 的 Ticket interface 中，`dryrun_output` 字段后面添加：

```typescript
export interface Ticket {
  // ... 现有字段 ...
  dryrun_output: string
  exec_output: string  // 新增
  // ...
}
```

- [ ] **Step 2: 提交**

```bash
git add web/src/types/index.ts
git commit -m "feat: add exec_output to Ticket TypeScript type"
```

---

## Self-Review

**1. Spec 覆盖检查：**
- ✅ 执行命令回显 → formatExecReport 的 `# 命令` 段
- ✅ stderr 输出包含 → formatExecReport 的 `# stderr` 段
- ✅ dryrun_output 完整报告 → Task 3
- ✅ exec_output 真实执行报告 → Task 4
- ✅ AI handler 返回 dryrun_output → Task 5
- ✅ 前端类型更新 → Task 6

**2. Placeholder 扫描：** 无 TBD/TODO/模糊描述

**3. 类型一致性检查：**
- `formatExecReport(command string, stdout string, stderr string, exitCode int, result string) string` — 所有调用点参数类型匹配
- `ExecOutput string` — GORM model 和 JSON tag (`exec_output`) 一致
- TypeScript `exec_output: string` 与 JSON tag 对应
