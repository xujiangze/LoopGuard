# 程序注册改造：文件上传 + 去白名单 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将程序注册从 binary_path+params_schema 白名单模式改造为文件上传模式，工单提交 args 改为 []string。

**Architecture:** Program 模型去掉 BinaryPath/ParamsSchema，新增 EntryFile。文件存 workspace/{project}/{name}/。Service 层新增文件管理逻辑（保存/清理/路径拼接）。Handler 层改为 multipart/form-data。Ticket 的 Args 从 map 改为 []string，validateArgs 从白名单改为元字符检查。

**Tech Stack:** Go (Gin + GORM), React (TypeScript + Vite + Tailwind + shadcn/ui)

---

## Task 1: 更新 Program 数据模型

**Files:**
- Modify: `internal/model/models.go:32-45`
- Modify: `internal/model/models.go:7` (imports)

- [ ] **Step 1: 更新 Program 模型**

将 BinaryPath → EntryFile，删除 ParamsSchema，新增 UpdatedAt：

```go
package model

import (
	"time"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         Role      `gorm:"type:varchar(16);not null;default:user" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type APIKey struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	KeyHash   string    `gorm:"size:255;uniqueIndex;not null" json:"-"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type Program struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	Project        string    `gorm:"size:128;not null;uniqueIndex:uk_project_name" json:"project"`
	Name           string    `gorm:"size:128;not null;uniqueIndex:uk_project_name" json:"name"`
	EntryFile      string    `gorm:"size:256;not null" json:"entry_file"`
	Interpreter    string    `gorm:"size:256;default:''" json:"interpreter"`
	HelpText       string    `gorm:"type:text" json:"help_text"`
	ApproverID     uint64    `gorm:"not null" json:"approver_id"`
	TimeoutSec     int       `gorm:"not null;default:300" json:"timeout_sec"`
	SupportsDryrun bool      `gorm:"not null;default:true" json:"supports_dryrun"`
	Enabled        bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

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

type ExecKind string

const (
	ExecKindDryrun ExecKind = "dryrun"
	ExecKindReal   ExecKind = "real"
)

type Execution struct {
	ID         uint64     `gorm:"primaryKey" json:"id"`
	TicketID   uint64     `gorm:"not null;index" json:"ticket_id"`
	Kind       ExecKind   `gorm:"type:varchar(16);not null" json:"kind"`
	Command    string     `gorm:"size:2048;not null" json:"command"`
	ExitCode   int        `json:"exit_code"`
	Stdout     string     `gorm:"type:mediumtext" json:"stdout"`
	Stderr     string     `gorm:"type:mediumtext" json:"stderr"`
	DurationMs int        `json:"duration_ms"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}
```

注意：需要添加 `"gorm.io/datatypes"` import，因为 Ticket.Args 仍然使用 datatypes.JSON。

- [ ] **Step 2: 运行测试确认编译通过**

Run: `go build ./...`
Expected: 编译失败（因为引用 BinaryPath/ParamsSchema 的代码还未更新），确认模型变更正确

- [ ] **Step 3: 提交**

```bash
git add internal/model/models.go
git commit -m "refactor: Program 模型 BinaryPath→EntryFile, 删除 ParamsSchema, 新增 UpdatedAt"
```

---

## Task 2: 更新 ProgramService（Register + 新增 Update + 文件管理）

**Files:**
- Modify: `internal/service/program.go`
- Create: `internal/service/upload.go` (文件管理工具函数)

- [ ] **Step 1: 创建文件管理工具函数**

创建 `internal/service/upload.go`：

```go
package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gorm.io/datatypes"
)

const (
	maxFileSize   = 10 << 20 // 10MB
	maxFileCount  = 20
)

var safeNameRe = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
var dangerousArgRe = regexp.MustCompile("[;&|`$(){}><!]")

func validateProjectName(s string) error {
	if !safeNameRe.MatchString(s) {
		return fmt.Errorf("%s 包含非法字符，只允许字母数字下划线短横线", s)
	}
	return nil
}

func sanitizeFilename(name string) error {
	if name == "" {
		return errors.New("文件名不能为空")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") || strings.Contains(name, "\x00") {
		return fmt.Errorf("文件名 %q 包含非法字符", name)
	}
	return nil
}

func programDir(workspaceDir, project, name string) string {
	return filepath.Join(workspaceDir, project, name)
}

func saveUploadedFiles(dir string, files []*multipart.FileHeader, entryFile string) error {
	if len(files) > maxFileCount {
		return fmt.Errorf("文件数量超过上限 %d", maxFileCount)
	}
	found := false
	for _, fh := range files {
		if fh.Filename == entryFile {
			found = true
		}
		if err := sanitizeFilename(fh.Filename); err != nil {
			return err
		}
		if fh.Size > maxFileSize {
			return fmt.Errorf("文件 %s 超过大小上限 10MB", fh.Filename)
		}
	}
	if !found {
		return fmt.Errorf("入口文件 %s 不在上传文件中", entryFile)
	}

	// 全量覆盖：先删旧目录再重建
	os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	for _, fh := range files {
		src, err := fh.Open()
		if err != nil {
			return fmt.Errorf("读取上传文件 %s 失败: %w", fh.Filename, err)
		}
		dstPath := filepath.Join(dir, fh.Filename)
		dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			src.Close()
			return fmt.Errorf("创建文件 %s 失败: %w", fh.Filename, err)
		}
		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()
		if err != nil {
			return fmt.Errorf("写入文件 %s 失败: %w", fh.Filename, err)
		}
	}
	return nil
}

func ValidateArgs(args []string) error {
	for _, arg := range args {
		if arg == "--only-print" {
			return errors.New("--only-print 为系统保留参数，禁止传入")
		}
		if dangerousArgRe.MatchString(arg) {
			return fmt.Errorf("参数包含危险字符: %s", arg)
		}
	}
	return nil
}
```

- [ ] **Step 2: 重写 ProgramService**

重写 `internal/service/program.go`：

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"mime/multipart"
)

type ProgramService struct {
	store        *store.Store
	exec         executor.Executor
	workspaceDir string
}

func NewProgramService(s *store.Store, e executor.Executor, workspaceDir string) *ProgramService {
	return &ProgramService{store: s, exec: e, workspaceDir: workspaceDir}
}

type RegisterInput struct {
	Project     string
	Name        string
	EntryFile   string
	Interpreter string
	ApproverID  uint64
	TimeoutSec  int
	Files       []*multipart.FileHeader
}

func (svc *ProgramService) Register(ctx context.Context, in RegisterInput) (*model.Program, error) {
	if in.Project == "" || in.Name == "" || in.EntryFile == "" {
		return nil, errors.New("project/name/entry_file 必填")
	}
	if err := validateProjectName(in.Project); err != nil {
		return nil, err
	}
	if err := validateProjectName(in.Name); err != nil {
		return nil, err
	}
	if in.Interpreter == "" {
		return nil, errors.New("interpreter 必填")
	}
	if len(in.Files) == 0 {
		return nil, errors.New("至少上传一个文件")
	}
	if _, err := svc.store.GetUser(in.ApproverID); err != nil {
		return nil, errors.New("审批人不存在")
	}

	dir := programDir(svc.workspaceDir, in.Project, in.Name)
	if err := saveUploadedFiles(dir, in.Files, in.EntryFile); err != nil {
		return nil, err
	}

	binaryPath := filepath.Join(dir, in.EntryFile)
	help, err := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: binaryPath, Interpreter: in.Interpreter, Args: []string{"--help"}, TimeoutSec: 10, WorkDir: dir,
	})
	if err != nil {
		return nil, errors.New("无法执行 --help：" + err.Error())
	}
	combined := help.Stdout + "\n" + help.Stderr
	if strings.Contains(strings.ToLower(combined), "unknown flag: --only-print") ||
		strings.Contains(strings.ToLower(combined), "unknown flag --only-print") {
		return nil, errors.New("程序未识别 --only-print 参数，拒绝注册")
	}

	timeout := in.TimeoutSec
	if timeout <= 0 {
		timeout = 300
	}
	p := &model.Program{
		Project: in.Project, Name: in.Name, EntryFile: in.EntryFile,
		Interpreter: in.Interpreter, HelpText: combined,
		ApproverID: in.ApproverID, TimeoutSec: timeout,
		SupportsDryrun: true, Enabled: true,
	}
	if err := svc.store.CreateProgram(p); err != nil {
		return nil, err
	}
	return p, nil
}

type UpdateInput struct {
	EntryFile   *string
	Interpreter *string
	ApproverID  *uint64
	TimeoutSec  *int
	Enabled     *bool
	Files       []*multipart.FileHeader
}

func (svc *ProgramService) Update(ctx context.Context, id uint64, in UpdateInput) (*model.Program, error) {
	p, err := svc.store.GetProgram(id)
	if err != nil {
		return nil, errors.New("程序不存在")
	}

	dir := programDir(svc.workspaceDir, p.Project, p.Name)
	needRefreshHelp := false

	if len(in.Files) > 0 {
		entryFile := p.EntryFile
		if in.EntryFile != nil {
			entryFile = *in.EntryFile
		}
		if err := saveUploadedFiles(dir, in.Files, entryFile); err != nil {
			return nil, err
		}
		p.EntryFile = entryFile
		needRefreshHelp = true
	} else if in.EntryFile != nil {
		p.EntryFile = *in.EntryFile
		needRefreshHelp = true
	}

	if in.Interpreter != nil {
		p.Interpreter = *in.Interpreter
		needRefreshHelp = true
	}
	if in.ApproverID != nil {
		if _, err := svc.store.GetUser(*in.ApproverID); err != nil {
			return nil, errors.New("审批人不存在")
		}
		p.ApproverID = *in.ApproverID
	}
	if in.TimeoutSec != nil {
		if *in.TimeoutSec > 0 {
			p.TimeoutSec = *in.TimeoutSec
		}
	}
	if in.Enabled != nil {
		p.Enabled = *in.Enabled
	}

	if needRefreshHelp {
		binaryPath := filepath.Join(dir, p.EntryFile)
		help, err := svc.exec.Run(ctx, executor.ExecRequest{
			BinaryPath: binaryPath, Interpreter: p.Interpreter, Args: []string{"--help"}, TimeoutSec: 10, WorkDir: dir,
		})
		if err == nil && help != nil {
			p.HelpText = help.Stdout + "\n" + help.Stderr
		}
	}

	if err := svc.store.UpdateProgram(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (svc *ProgramService) List() ([]model.Program, error) { return svc.store.ListPrograms() }

func (svc *ProgramService) ProgramPath(p *model.Program) string {
	return filepath.Join(svc.workspaceDir, p.Project, p.Name, p.EntryFile)
}

func (svc *ProgramService) ProgramWorkDir(p *model.Program) string {
	return filepath.Join(svc.workspaceDir, p.Project, p.Name)
}
```

- [ ] **Step 3: 更新 store 中 DeleteProgram 辅助（如需要）**

在 `internal/store/store.go` 中 Programs 部分添加 DeleteProgram：

```go
func (s *Store) DeleteProgram(id uint64) error { return s.db.Delete(&model.Program{}, id).Error }
```

- [ ] **Step 4: 提交**

```bash
git add internal/service/program.go internal/service/upload.go internal/store/store.go
git commit -m "feat: ProgramService 文件上传注册/更新，新增 upload.go 工具函数"
```

---

## Task 3: 重写 TicketService（args []string + 元字符检查）

**Files:**
- Modify: `internal/service/ticket.go`

- [ ] **Step 1: 重写 ticket.go**

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"gorm.io/datatypes"
)

type TicketService struct {
	store        *store.Store
	exec         executor.Executor
	workspaceDir string
}

func NewTicketService(s *store.Store, e executor.Executor, workspaceDir string) *TicketService {
	return &TicketService{store: s, exec: e, workspaceDir: workspaceDir}
}

type SubmitInput struct {
	Project  string
	Name     string
	APIKeyID uint64
	Args     []string
}

func (svc *TicketService) Submit(ctx context.Context, in SubmitInput) (*model.Ticket, error) {
	p, err := svc.store.GetProgramByProjectName(in.Project, in.Name)
	if err != nil {
		return nil, errors.New("程序未注册：" + in.Project + "/" + in.Name)
	}
	if !p.Enabled {
		return nil, errors.New("程序已禁用")
	}
	if err := ValidateArgs(in.Args); err != nil {
		return nil, err
	}

	argsJSON, _ := json.Marshal(in.Args)
	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(argsJSON),
		Status: model.StatusPendingDryrun, SubmittedBy: in.APIKeyID,
		ApproverID: p.ApproverID,
	}
	if err := svc.store.CreateTicket(tk); err != nil {
		return nil, err
	}

	binaryPath := filepath.Join(svc.workspaceDir, p.Project, p.Name, p.EntryFile)
	workDir := filepath.Join(svc.workspaceDir, p.Project, p.Name)
	res, runErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: binaryPath, Interpreter: p.Interpreter, Args: in.Args, DryRun: true,
		TimeoutSec: p.TimeoutSec, WorkDir: workDir,
	})
	now := time.Now()
	exe := &model.Execution{TicketID: tk.ID, Kind: model.ExecKindDryrun, StartedAt: &now}
	if res != nil {
		exe.Command = res.Command
		exe.ExitCode = res.ExitCode
		exe.Stdout = res.Stdout
		exe.Stderr = res.Stderr
		exe.DurationMs = int(res.DurationMs)
	}
	fin := time.Now()
	exe.FinishedAt = &fin
	_ = svc.store.CreateExecution(exe)

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
}

func (svc *TicketService) Get(id uint64) (*model.Ticket, error) { return svc.store.GetTicket(id) }

func errString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

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
	fmt.Fprintf(&sb, "退出码: %d | %s", exitCode, result)
	return sb.String()
}

func (svc *TicketService) Approve(ctx context.Context, ticketID, userID uint64) (*model.Ticket, error) {
	tk, err := svc.store.GetTicket(ticketID)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	if tk.ApproverID != userID {
		return nil, errors.New("无权审批：你不是该工单的指定审批人")
	}
	if !model.CanTransition(tk.Status, model.StatusApproved) {
		return nil, errors.New("当前状态不可审批：" + string(tk.Status))
	}

	now := time.Now()
	tk.Status = model.StatusApproved
	tk.ApprovedBy = &userID
	tk.ApprovedAt = &now
	_ = svc.store.UpdateTicket(tk)

	p, err := svc.store.GetProgram(tk.ProgramID)
	if err != nil {
		return nil, err
	}

	tk.Status = model.StatusExecuting
	_ = svc.store.UpdateTicket(tk)

	var args []string
	_ = json.Unmarshal(tk.Args, &args)

	binaryPath := filepath.Join(svc.workspaceDir, p.Project, p.Name, p.EntryFile)
	workDir := filepath.Join(svc.workspaceDir, p.Project, p.Name)
	res, runErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: binaryPath, Interpreter: p.Interpreter, Args: args, DryRun: false,
		TimeoutSec: p.TimeoutSec, WorkDir: workDir,
	})

	start := now
	exe := &model.Execution{TicketID: tk.ID, Kind: model.ExecKindReal, StartedAt: &start}
	if res != nil {
		exe.Command = res.Command
		exe.ExitCode = res.ExitCode
		exe.Stdout = res.Stdout
		exe.Stderr = res.Stderr
		exe.DurationMs = int(res.DurationMs)
	}
	fin := time.Now()
	exe.FinishedAt = &fin
	_ = svc.store.CreateExecution(exe)

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
}

func (svc *TicketService) Reject(ticketID, userID uint64, reason string) (*model.Ticket, error) {
	tk, err := svc.store.GetTicket(ticketID)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	if tk.ApproverID != userID {
		return nil, errors.New("无权操作：你不是该工单的指定审批人")
	}
	if !model.CanTransition(tk.Status, model.StatusRejected) {
		return nil, errors.New("当前状态不可驳回：" + string(tk.Status))
	}
	tk.Status = model.StatusRejected
	tk.RejectReason = reason
	_ = svc.store.UpdateTicket(tk)
	return tk, nil
}
```

注意：需要添加 `"path/filepath"` 到 imports。

- [ ] **Step 2: 提交**

```bash
git add internal/service/ticket.go
git commit -m "refactor: TicketService args 改为 []string，路径从 workspace 拼接"
```

---

## Task 4: 更新 API Handler 层

**Files:**
- Modify: `internal/api/admin_handler.go`
- Modify: `internal/api/ai_handler.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: 重写 admin_handler.go**

AdminHandler 需要 store + programs + workspaceDir：

```go
package api

import (
	"net/http"
	"strconv"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	store        *store.Store
	programs     *service.ProgramService
	workspaceDir string
}

func NewAdminHandler(s *store.Store, p *service.ProgramService, workspaceDir string) *AdminHandler {
	return &AdminHandler{store: s, programs: p, workspaceDir: workspaceDir}
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min:6"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := model.RoleUser
	if req.Role == "admin" {
		role = model.RoleAdmin
	}
	hash, _ := auth.HashPassword(req.Password)
	u := &model.User{Username: req.Username, PasswordHash: hash, Role: role}
	if err := h.store.CreateUser(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "username": u.Username, "role": u.Role})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []model.User
	if err := h.store.DB().Order("id asc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) ListAPIKeys(c *gin.Context) {
	var keys []model.APIKey
	if err := h.store.DB().Order("id desc").Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, keys)
}

func (h *AdminHandler) CreateAPIKey(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plain := auth.GenerateAPIKey()
	k := &model.APIKey{Name: req.Name, KeyHash: auth.HashAPIKey(plain), Enabled: true}
	if err := h.store.CreateAPIKey(k); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": k.ID, "name": k.Name, "api_key": plain})
}

func (h *AdminHandler) CreateProgram(c *gin.Context) {
	project := c.PostForm("project")
	name := c.PostForm("name")
	entryFile := c.PostForm("entry_file")
	interpreter := c.PostForm("interpreter")
	approverIDStr := c.PostForm("approver_id")
	timeoutSecStr := c.PostForm("timeout_sec")

	if project == "" || name == "" || entryFile == "" || interpreter == "" || approverIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project/name/entry_file/interpreter/approver_id 必填"})
		return
	}
	approverID, _ := strconv.ParseUint(approverIDStr, 10, 64)
	timeoutSec, _ := strconv.Atoi(timeoutSecStr)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart 表单解析失败"})
		return
	}
	files := form.File["files"]

	p, err := h.programs.Register(c.Request.Context(), service.RegisterInput{
		Project: project, Name: name, EntryFile: entryFile, Interpreter: interpreter,
		ApproverID: approverID, TimeoutSec: timeoutSec, Files: files,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *AdminHandler) ListPrograms(c *gin.Context) {
	ps, err := h.programs.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ps)
}

func (h *AdminHandler) UpdateProgram(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var in service.UpdateInput

	// 可选字段
	if v := c.PostForm("entry_file"); v != "" {
		in.EntryFile = &v
	}
	if v := c.PostForm("interpreter"); v != "" {
		in.Interpreter = &v
	}
	if v := c.PostForm("approver_id"); v != "" {
		n, _ := strconv.ParseUint(v, 10, 64)
		in.ApproverID = &n
	}
	if v := c.PostForm("timeout_sec"); v != "" {
		n, _ := strconv.Atoi(v)
		in.TimeoutSec = &n
	}
	if v := c.PostForm("enabled"); v != "" {
		b := v == "true"
		in.Enabled = &b
	}

	form, err := c.MultipartForm()
	if err == nil {
		in.Files = form.File["files"]
	}

	p, err := h.programs.Update(c.Request.Context(), id, in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *AdminHandler) UpdateAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var k model.APIKey
	if err := h.store.DB().First(&k, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API Key 不存在"})
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Enabled != nil {
		k.Enabled = *req.Enabled
	}
	if err := h.store.UpdateAPIKey(&k); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, k)
}

func (h *AdminHandler) DeleteAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.store.DeleteAPIKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (h *AdminHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Password string `json:"password" binding:"required,min:6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, _ := auth.HashPassword(req.Password)
	if err := h.store.UpdateUserPassword(id, hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码已重置"})
}
```

- [ ] **Step 2: 更新 ai_handler.go**

```go
package api

import (
	"fmt"
	"net/http"
	"strconv"

	"LoopGuard/internal/config"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	tickets *service.TicketService
	cfg     config.Config
}

func NewAIHandler(t *service.TicketService, cfg config.Config) *AIHandler {
	return &AIHandler{tickets: t, cfg: cfg}
}

type submitReq struct {
	Project string   `json:"project" binding:"required"`
	Name    string   `json:"name" binding:"required"`
	Args    []string `json:"args"`
}

func (h *AIHandler) Submit(c *gin.Context) {
	var req submitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	apiKeyID := c.GetUint64("api_key_id")
	tk, err := h.tickets.Submit(c.Request.Context(), service.SubmitInput{
		Project: req.Project, Name: req.Name, APIKeyID: apiKeyID, Args: req.Args,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	url := fmt.Sprintf("%s/tickets/%d", h.cfg.BaseURL, tk.ID)
	resp := gin.H{"ticket_id": tk.ID, "status": tk.Status, "approval_url": url}
	switch tk.Status {
	case model.StatusPendingApproval:
		resp["next_action"] = fmt.Sprintf(
			"任务已提交，需人工审批。请通知用户访问审批链接 %s 找审批人审批。审批通过后任务将自动执行，你可轮询 GET /api/v1/tickets/%d 获取结果。",
			url, tk.ID)
	case model.StatusDryrunFailed:
		resp["next_action"] = "dry-run 校验未通过，任务未进入审批。请检查程序是否正确实现 --only-print（需输出 DRYRUN-OK 且退出码为 0）。详情见 dryrun_output。"
		resp["dryrun_output"] = tk.DryrunOutput
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AIHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 id"})
		return
	}
	tk, err := h.tickets.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	c.JSON(http.StatusOK, tk)
}
```

- [ ] **Step 3: 更新 router.go**

需要更新 Deps、NewAdminHandler 调用、ProgramService 构造、路由新增 AI 访问 programs：

```go
package api

import (
	"io/fs"
	"net/http"
	"time"

	"LoopGuard/internal/config"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"
	"LoopGuard/web"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Deps struct {
	Store      *store.Store
	TicketSvc  *service.TicketService
	ProgramSvc *service.ProgramService
	Cfg        config.Config
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	ai := NewAIHandler(d.TicketSvc, d.Cfg)
	human := NewHumanHandler(d.Store, d.TicketSvc, d.Cfg)
	admin := NewAdminHandler(d.Store, d.ProgramSvc, d.Cfg.WorkspaceDir)

	v1 := r.Group("/api/v1")

	v1.POST("/auth/login", human.Login)

	aiGrp := v1.Group("", APIKeyAuth(d.Store))
	aiGrp.POST("/tickets", ai.Submit)

	// AI 也可获取程序列表（含 HelpText）
	aiGrp.GET("/programs", admin.ListPrograms)

	// GET /tickets/:id: AI 轮询和人工查看共用，接受 API Key 或 JWT
	v1.GET("/tickets/:id", APIKeyOrJWTAuth(d.Store, d.Cfg.JWTSecret), ai.Get)
	v1.GET("/tickets/:id/executions", JWTAuth(d.Cfg.JWTSecret), human.ListExecutions)

	jwtGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret))
	jwtGrp.GET("/tickets", human.ListMine)
	jwtGrp.POST("/tickets/:id/approve", human.Approve)
	jwtGrp.POST("/tickets/:id/reject", human.Reject)

	adminGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret), AdminOnly())
	adminGrp.POST("/programs", admin.CreateProgram)
	adminGrp.GET("/programs", admin.ListPrograms)
	adminGrp.PUT("/programs/:id", admin.UpdateProgram)
	adminGrp.POST("/users", admin.CreateUser)
	adminGrp.GET("/users", admin.ListUsers)
	adminGrp.POST("/api-keys", admin.CreateAPIKey)
	adminGrp.GET("/api-keys", admin.ListAPIKeys)
	adminGrp.PUT("/api-keys/:id", admin.UpdateAPIKey)
	adminGrp.DELETE("/api-keys/:id", admin.DeleteAPIKey)
	adminGrp.PUT("/users/:id/password", admin.ResetPassword)

	// 前端静态文件服务
	distFS, _ := fs.Sub(web.DistFS, "dist")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		f, err := distFS.(fs.ReadFileFS).ReadFile(path[1:])
		if err != nil {
			idx, _ := distFS.(fs.ReadFileFS).ReadFile("index.html")
			c.Data(http.StatusOK, "text/html; charset=utf-8", idx)
			return
		}
		c.Data(http.StatusOK, mimeByExt(path), f)
	})

	return r
}

func mimeByExt(path string) string {
	switch {
	case len(path) > 3 && path[len(path)-3:] == ".js":
		return "application/javascript"
	case len(path) > 4 && path[len(path)-4:] == ".css":
		return "text/css"
	case len(path) > 5 && path[len(path)-5:] == ".html":
		return "text/html; charset=utf-8"
	case len(path) > 4 && path[len(path)-4:] == ".svg":
		return "image/svg+xml"
	case len(path) > 4 && path[len(path)-4:] == ".png":
		return "image/png"
	case len(path) > 4 && path[len(path)-4:] == ".ico":
		return "image/x-icon"
	case len(path) > 4 && path[len(path)-4:] == ".wasm":
		return "application/wasm"
	case len(path) > 5 && path[len(path)-5:] == ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
```

- [ ] **Step 4: 更新 CLI 层的 ProgramService 构造**

需要更新 `internal/cli/serve.go` 中 NewProgramService 和 NewTicketService 调用，传入 workspaceDir。先读文件查看当前代码，然后更新：

```go
// serve.go 中的相关行需要从：
programSvc := service.NewProgramService(store, exec)
ticketSvc := service.NewTicketService(store, exec)
// 改为：
programSvc := service.NewProgramService(store, exec, cfg.WorkspaceDir)
ticketSvc := service.NewTicketService(store, exec, cfg.WorkspaceDir)
```

- [ ] **Step 5: 提交**

```bash
git add internal/api/admin_handler.go internal/api/ai_handler.go internal/api/router.go internal/cli/serve.go
git commit -m "feat: API 层改为 multipart 上传，AI 可获取程序列表，args 改为 []string"
```

---

## Task 5: 更新测试

**Files:**
- Modify: `internal/service/ticket_test.go`
- Modify: `internal/service/program_test.go`
- Create: `internal/service/upload_test.go`

- [ ] **Step 1: 创建 upload_test.go**

```go
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid lowercase", "my-project", false},
		{"valid with numbers", "project123", false},
		{"valid with underscore", "my_project", false},
		{"valid mixed", "My-Project_1", false},
		{"rejects slash", "my/project", true},
		{"rejects dot", "my.project", true},
		{"rejects space", "my project", true},
		{"rejects empty", "", true},
		{"rejects chinese", "项目", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid py file", "entry.py", false},
		{"valid with underscore", "my_script.py", false},
		{"rejects path traversal", "../../../etc/passwd", true},
		{"rejects absolute path", "/etc/passwd", true},
		{"rejects backslash", "dir\\file.py", true},
		{"rejects null byte", "file\x00.py", true},
		{"rejects empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizeFilename(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateArgsRejectsOnlyPrint(t *testing.T) {
	err := ValidateArgs([]string{"--only-print"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "保留")
}

func TestValidateArgsRejectsDangerousChars(t *testing.T) {
	dangerous := []string{";rm -rf", "|cat /etc/passwd", "&whoami", "`id`", "$(whoami)", "{bad}", ">file", "<file", "!bang"}
	for _, arg := range dangerous {
		err := ValidateArgs([]string{arg})
		require.Error(t, err, "expected error for arg: %s", arg)
		assert.Contains(t, err.Error(), "危险字符")
	}
}

func TestValidateArgsAcceptsNormal(t *testing.T) {
	err := ValidateArgs([]string{"-m", "group_unban", "-e", "never", "--skip-confirm"})
	require.NoError(t, err)
}
```

- [ ] **Step 2: 重写 ticket_test.go**

```go
package service

import (
	"context"
	"errors"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTicketService(t *testing.T, exec executor.Executor) (*TicketService, *store.Store) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=private"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return NewTicketService(s, exec, "/tmp/test-workspace"), s
}

func seedProgram(t *testing.T, s *store.Store) *model.Program {
	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	p := &model.Program{Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 30, SupportsDryrun: true, Enabled: true}
	require.NoError(t, s.CreateProgram(p))
	return p
}

func TestSubmitTicketDryrunPass(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod --only-print",
		ExitCode: 0, Stdout: "DRYRUN-OK\nwill deploy", Stderr: "",
	}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "# 命令")
	assert.Contains(t, tk.DryrunOutput, "DRYRUN-OK")
	assert.Contains(t, tk.DryrunOutput, "校验: 通过")
}

func TestSubmitTicketDryrunFail(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod --only-print",
		ExitCode: 0, Stdout: "no marker", Stderr: "",
	}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDryrunFailed, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "# 命令")
	assert.Contains(t, tk.DryrunOutput, "校验: 失败")
}

func TestSubmitRejectsDangerousArg(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	_, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod; rm -rf /"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "危险字符")
}

func TestSubmitRejectsOnlyPrintInArgs(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	_, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"--only-print"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "保留")
}

func TestSubmitWithInterpreter(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK\npython output"}}
	svc, s := newTicketService(t, fe)
	prog := seedProgram(t, s)
	prog.Interpreter = "python3"
	require.NoError(t, s.UpdateProgram(prog))

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
}

func TestFormatExecReport(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		stdout   string
		stderr   string
		exitCode int
		result   string
		want     []string
	}{
		{
			name:     "dryrun pass with output",
			command:  "python3 deploy.sh -e prod --only-print",
			stdout:   "将执行: kubectl apply\nDRYRUN-OK",
			stderr:   "",
			exitCode: 0,
			result:   "校验: 通过",
			want: []string{
				"# 命令\npython3 deploy.sh -e prod --only-print",
				"# stdout\n将执行: kubectl apply\nDRYRUN-OK",
				"# stderr\n(无)",
				"# 结果\n退出码: 0 | 校验: 通过",
			},
		},
		{
			name:     "exec fail with stderr",
			command:  "bash deploy.sh -e prod",
			stdout:   "",
			stderr:   "error: connection refused",
			exitCode: 1,
			result:   "校验: 失败 - dry-run 退出码非 0（实际 1）",
			want: []string{
				"# 命令\nbash deploy.sh -e prod",
				"# stdout\n",
				"# stderr\nerror: connection refused",
				"# 结果\n退出码: 1 | 校验: 失败 - dry-run 退出码非 0（实际 1）",
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

func TestSubmitTicketDryrunRunError(t *testing.T) {
	fe := &fakeExecutor{result: nil, err: errors.New("binary not found")}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDryrunFailed, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "# 命令")
	assert.Contains(t, tk.DryrunOutput, "binary not found")
}

func TestApproveFillsExecOutput(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod", ExitCode: 0,
		Stdout: "deployment configured", Stderr: "", DurationMs: 1523,
	}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s)

	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(`["-e","prod"]`),
		Status: model.StatusPendingApproval, SubmittedBy: 7,
		ApproverID: p.ApproverID,
		DryrunOutput: "# 命令\nbash deploy.sh --only-print\n\n# 结果\n退出码: 0 | 校验: 通过",
	}
	require.NoError(t, s.CreateTicket(tk))

	result, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusDone, result.Status)
	assert.Contains(t, result.ExecOutput, "# 命令")
	assert.Contains(t, result.ExecOutput, "deployment configured")
	assert.Contains(t, result.ExecOutput, "耗时: 1523ms")
}

func TestApproveExecFailedFillsExecOutput(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod", ExitCode: 1,
		Stdout: "", Stderr: "error: connection refused", DurationMs: 500,
	}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s)

	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(`["-e","prod"]`),
		Status: model.StatusPendingApproval, SubmittedBy: 7,
		ApproverID: p.ApproverID,
		DryrunOutput: "# 命令\nbash deploy.sh --only-print",
	}
	require.NoError(t, s.CreateTicket(tk))

	result, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusExecFailed, result.Status)
	assert.Contains(t, result.ExecOutput, "# stderr\nerror: connection refused")
	assert.Contains(t, result.ExecOutput, "退出码: 1")
}
```

- [ ] **Step 3: 重写 program_test.go**

```go
package service

import (
	"context"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeExecutor struct {
	result *executor.ExecResult
	err    error
}

func (f *fakeExecutor) Run(_ context.Context, _ executor.ExecRequest) (*executor.ExecResult, error) {
	return f.result, f.err
}

func newTestProgramService(t *testing.T, exec executor.Executor) (*ProgramService, *store.Store) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return NewProgramService(s, exec, "/tmp/test-workspace"), s
}

func TestRegisterProgramValidation(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s := newTestProgramService(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))

	// 缺少 interpreter 应失败
	_, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		ApproverID: u.ID, TimeoutSec: 60,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "interpreter")
}

func TestRegisterProgramRejectsBadProjectName(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage"}}
	svc, _ := newTestProgramService(t, fe)

	_, err := svc.Register(context.Background(), RegisterInput{
		Project: "my/project", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "非法字符")
}
```

- [ ] **Step 4: 运行全部测试**

Run: `go test ./...`
Expected: 所有测试通过

- [ ] **Step 5: 提交**

```bash
git add internal/service/upload_test.go internal/service/ticket_test.go internal/service/program_test.go
git commit -m "test: 更新测试适配新模型和 []string args"
```

---

## Task 6: 更新前端

**Files:**
- Modify: `web/src/types/index.ts`
- Modify: `web/src/pages/ProgramPage.tsx`
- Modify: `web/src/lib/api.ts`

- [ ] **Step 1: 更新 types/index.ts**

将 Program 接口中的 binary_path → entry_file，删除 params_schema，新增 updated_at。Ticket 的 args 改为 string[]：

```typescript
export type Role = "user" | "admin"

export type TicketStatus =
  | "pending_dryrun"
  | "dryrun_failed"
  | "pending_approval"
  | "approved"
  | "executing"
  | "done"
  | "exec_failed"
  | "rejected"

export interface User {
  id: number
  username: string
  role: Role
  created_at: string
}

export interface Ticket {
  id: number
  program_id: number
  args: string[]
  status: TicketStatus
  submitted_by: number
  approver_id: number
  dryrun_output: string
  exec_output: string
  approved_by: number | null
  approved_at: string | null
  reject_reason: string
  created_at: string
  updated_at: string
}

export interface Program {
  id: number
  project: string
  name: string
  entry_file: string
  interpreter: string
  help_text: string
  approver_id: number
  timeout_sec: number
  supports_dryrun: boolean
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface APIKey {
  id: number
  name: string
  enabled: boolean
  created_at: string
}

export interface Execution {
  id: number
  ticket_id: number
  kind: "dryrun" | "real"
  command: string
  exit_code: number
  stdout: string
  stderr: string
  duration_ms: number
  started_at: string | null
  finished_at: string | null
}

export const TICKET_STATUS_LABELS: Record<TicketStatus, string> = {
  pending_dryrun: "Dry-run 中",
  dryrun_failed: "Dry-run 失败",
  pending_approval: "待审批",
  approved: "已批准",
  executing: "执行中",
  done: "已完成",
  exec_failed: "执行失败",
  rejected: "已驳回",
}
```

- [ ] **Step 2: 给 api.ts 添加 upload 方法**

在 api 对象中添加 upload 方法用于 multipart/form-data 上传：

```typescript
import { getToken, clearToken } from "./auth"

const BASE_URL = "/api/v1"

export class ApiError extends Error {
  status: number
  constructor(
    status: number,
    message: string,
  ) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    ...(init?.headers as Record<string, string>),
  }
  // upload 方法自己设 Content-Type，其余默认 JSON
  if (!init?.headers || !("Content-Type" in (init?.headers as Record<string, string>))) {
    if (init?.body && !(init?.body instanceof FormData)) {
      headers["Content-Type"] = "application/json"
    }
  }
  if (token) {
    headers["Authorization"] = `Bearer ${token}`
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...init, headers })

  if (res.status === 401) {
    clearToken()
    window.location.hash = "#/login"
    throw new ApiError(401, "登录已过期，请重新登录")
  }

  const data = await res.json()
  if (!res.ok) {
    throw new ApiError(res.status, data.error || "请求失败")
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),

  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }),

  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PUT", body: body ? JSON.stringify(body) : undefined }),

  del: <T>(path: string) => request<T>(path, { method: "DELETE" }),

  upload: <T>(path: string, formData: FormData) =>
    request<T>(path, { method: "POST", body: formData }),

  uploadPut: <T>(path: string, formData: FormData) =>
    request<T>(path, { method: "PUT", body: formData }),
}
```

- [ ] **Step 3: 重写 ProgramPage.tsx**

```tsx
import { useEffect, useRef, useState } from "react"
import { api } from "@/lib/api"
import type { Program, User } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { toast } from "sonner"

export function ProgramPage() {
  const [programs, setPrograms] = useState<Program[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Program | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const [form, setForm] = useState({
    project: "",
    name: "",
    entry_file: "",
    interpreter: "",
    approver_id: "",
    timeout_sec: "300",
  })
  const [createFiles, setCreateFiles] = useState<FileList | null>(null)

  const [editForm, setEditForm] = useState({
    enabled: true,
    interpreter: "",
    approver_id: "",
    timeout_sec: "300",
  })
  const [editFiles, setEditFiles] = useState<FileList | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const editFileInputRef = useRef<HTMLInputElement>(null)

  const fetchData = () => {
    setLoading(true)
    Promise.all([api.get<Program[]>("/programs"), api.get<User[]>("/users")])
      .then(([ps, us]) => {
        setPrograms(ps)
        setUsers(us)
      })
      .catch(() => {
        setPrograms([])
        setUsers([])
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => { fetchData() }, [])

  const userName = (id: number) => users.find((u) => u.id === id)?.username ?? String(id)
  const userInitial = (id: number) => {
    const name = userName(id)
    return name === String(id) ? "?" : name.charAt(0).toUpperCase()
  }

  const handleCreate = async () => {
    if (!form.project || !form.name || !form.entry_file || !form.interpreter || !form.approver_id) {
      toast.error("请填写所有必填字段")
      return
    }
    if (!createFiles || createFiles.length === 0) {
      toast.error("请上传文件")
      return
    }

    const fd = new FormData()
    fd.append("project", form.project)
    fd.append("name", form.name)
    fd.append("entry_file", form.entry_file)
    fd.append("interpreter", form.interpreter)
    fd.append("approver_id", form.approver_id)
    fd.append("timeout_sec", form.timeout_sec)
    for (let i = 0; i < createFiles.length; i++) {
      fd.append("files", createFiles[i])
    }

    setSubmitting(true)
    try {
      await api.upload("/programs", fd)
      toast.success("程序注册成功", { description: `${form.project}/${form.name}` })
      setCreateOpen(false)
      setForm({ project: "", name: "", entry_file: "", interpreter: "", approver_id: "", timeout_sec: "300" })
      setCreateFiles(null)
      if (fileInputRef.current) fileInputRef.current.value = ""
      fetchData()
    } catch (err) {
      toast.error("创建失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setSubmitting(false)
    }
  }

  const openEdit = (p: Program) => {
    setEditTarget(p)
    setEditForm({ enabled: p.enabled, interpreter: p.interpreter, approver_id: String(p.approver_id), timeout_sec: String(p.timeout_sec) })
    setEditFiles(null)
    if (editFileInputRef.current) editFileInputRef.current.value = ""
    setEditOpen(true)
  }

  const handleEdit = async () => {
    if (!editTarget) return
    setSubmitting(true)
    try {
      const fd = new FormData()
      fd.append("enabled", String(editForm.enabled))
      if (editForm.interpreter) fd.append("interpreter", editForm.interpreter)
      if (editForm.approver_id) fd.append("approver_id", editForm.approver_id)
      if (editForm.timeout_sec) fd.append("timeout_sec", editForm.timeout_sec)
      if (editFiles) {
        for (let i = 0; i < editFiles.length; i++) {
          fd.append("files", editFiles[i])
        }
      }
      await api.uploadPut(`/programs/${editTarget.id}`, fd)
      toast.success("程序更新成功")
      setEditOpen(false)
      fetchData()
    } catch (err) {
      toast.error("更新失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setSubmitting(false)
    }
  }

  const toggleEnabled = async (p: Program) => {
    try {
      const fd = new FormData()
      fd.append("enabled", String(!p.enabled))
      await api.uploadPut(`/programs/${p.id}`, fd)
      toast.success(p.enabled ? "已禁用" : "已启用", { description: `${p.project}/${p.name}` })
      fetchData()
    } catch (err) {
      toast.error("操作失败", { description: err instanceof Error ? err.message : "未知错误" })
    }
  }

  if (loading) return <p className="text-muted-foreground text-sm">加载中...</p>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">程序管理</h1>
        <Button onClick={() => setCreateOpen(true)}>注册新程序</Button>
      </div>

      {programs.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center">暂无注册程序</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>项目 / 程序名</TableHead>
              <TableHead>入口文件</TableHead>
              <TableHead>审批人</TableHead>
              <TableHead className="w-20">超时</TableHead>
              <TableHead className="w-20">状态</TableHead>
              <TableHead className="w-28">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {programs.map((p) => (
              <TableRow key={p.id}>
                <TableCell>
                  <div className="font-mono font-medium">{p.project}/{p.name}</div>
                  <div className="text-xs text-muted-foreground mt-0.5">{p.interpreter}</div>
                </TableCell>
                <TableCell className="font-mono text-sm">{p.entry_file}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <div className="w-6 h-6 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-semibold">
                      {userInitial(p.approver_id)}
                    </div>
                    <span className="text-sm">{userName(p.approver_id)}</span>
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground">{p.timeout_sec}s</TableCell>
                <TableCell>
                  <Badge variant={p.enabled ? "default" : "outline"}>
                    {p.enabled ? "启用" : "禁用"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Button variant="ghost" size="sm" onClick={() => openEdit(p)}>编辑</Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className={p.enabled ? "text-destructive" : "text-green-600"}
                      onClick={() => toggleEnabled(p)}
                    >
                      {p.enabled ? "禁用" : "启用"}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* 创建弹窗 */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>注册新程序</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 max-h-[60vh] overflow-y-auto px-1">
            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">基本信息</p>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label>项目名 <span className="text-destructive">*</span></Label>
                  <Input placeholder="如 tsunami_ipban" value={form.project} onChange={(e) => setForm({ ...form, project: e.target.value })} />
                </div>
                <div className="space-y-1">
                  <Label>程序名 <span className="text-destructive">*</span></Label>
                  <Input placeholder="如 entry_ipban" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
                </div>
              </div>
              <div className="space-y-1">
                <Label>入口文件名 <span className="text-destructive">*</span></Label>
                <Input placeholder="如 entry_ipban.py" value={form.entry_file} onChange={(e) => setForm({ ...form, entry_file: e.target.value })} />
                <p className="text-xs text-muted-foreground">必须与上传文件中的文件名一致</p>
              </div>
              <div className="space-y-1">
                <Label>解释器 <span className="text-destructive">*</span></Label>
                <Input placeholder="如 python3、bash、node" value={form.interpreter} onChange={(e) => setForm({ ...form, interpreter: e.target.value })} />
              </div>
            </div>

            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">审批配置</p>
              <div className="space-y-1">
                <Label>审批人 <span className="text-destructive">*</span></Label>
                <Select value={form.approver_id} onValueChange={(v) => { if (v) setForm({ ...form, approver_id: v }) }}>
                  <SelectTrigger><SelectValue placeholder="选择审批人" /></SelectTrigger>
                  <SelectContent>
                    {users.map((u) => (
                      <SelectItem key={u.id} value={String(u.id)}>
                        {u.username}（{u.role}）
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label>超时时间</Label>
                <div className="flex items-center gap-2">
                  <Input type="number" className="w-24" value={form.timeout_sec} onChange={(e) => setForm({ ...form, timeout_sec: e.target.value })} />
                  <span className="text-sm text-muted-foreground">秒</span>
                </div>
              </div>
            </div>

            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">上传文件 <span className="text-destructive">*</span></p>
              <Input
                ref={fileInputRef}
                type="file"
                multiple
                onChange={(e) => setCreateFiles(e.target.files)}
              />
              {createFiles && (
                <p className="text-xs text-muted-foreground">
                  已选择 {createFiles.length} 个文件: {Array.from(createFiles).map((f) => f.name).join(", ")}
                </p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>取消</Button>
            <Button onClick={handleCreate} disabled={submitting}>
              {submitting ? "注册中..." : "注册"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 编辑弹窗 */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>编辑程序</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
              <div>
                <Label>启用状态</Label>
                <p className="text-xs text-muted-foreground mt-0.5">禁用后 AI 将无法提交此程序的工单</p>
              </div>
              <Switch
                checked={editForm.enabled}
                onCheckedChange={(checked: boolean) => setEditForm({ ...editForm, enabled: checked })}
              />
            </div>
            <div className="space-y-1">
              <Label>解释器</Label>
              <Input placeholder="如 python3" value={editForm.interpreter} onChange={(e) => setEditForm({ ...editForm, interpreter: e.target.value })} />
            </div>
            <div className="space-y-1">
              <Label>审批人</Label>
              <Select value={editForm.approver_id} onValueChange={(v) => { if (v) setEditForm({ ...editForm, approver_id: v }) }}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {users.map((u) => (
                    <SelectItem key={u.id} value={String(u.id)}>
                      {u.username}（{u.role}）
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label>超时（秒）</Label>
              <Input type="number" value={editForm.timeout_sec} onChange={(e) => setEditForm({ ...editForm, timeout_sec: e.target.value })} />
            </div>
            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">更新文件（可选）</p>
              <Input
                ref={editFileInputRef}
                type="file"
                multiple
                onChange={(e) => setEditFiles(e.target.files)}
              />
              {editFiles && (
                <p className="text-xs text-muted-foreground">
                  已选择 {editFiles.length} 个文件，上传后将替换全部现有文件
                </p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditOpen(false)}>取消</Button>
            <Button onClick={handleEdit} disabled={submitting}>
              {submitting ? "保存中..." : "保存"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
```

- [ ] **Step 4: 前端 lint 检查**

Run: `cd web && pnpm lint`
Expected: 无错误

- [ ] **Step 5: 提交**

```bash
git add web/src/types/index.ts web/src/lib/api.ts web/src/pages/ProgramPage.tsx
git commit -m "feat: 前端适配文件上传程序注册，去掉参数白名单"
```

---

## Task 7: 更新其他引用 BinaryPath 的测试和代码

**Files:**
- Check all `*_test.go` files for references to BinaryPath/ParamsSchema
- Modify: any files still referencing old fields

- [ ] **Step 1: 搜索并修复所有 BinaryPath/ParamsSchema 引用**

Run: `grep -rn "BinaryPath\|ParamsSchema\|params_schema\|binary_path" --include="*.go" internal/`
Expected: 无结果（所有引用已更新）

- [ ] **Step 2: 修复 store 相关测试（如有）**

Run: `grep -rn "BinaryPath\|ParamsSchema" --include="*_test.go" internal/`
Expected: 无结果

- [ ] **Step 3: 运行全部测试**

Run: `go test ./...`
Expected: 全部通过

- [ ] **Step 4: 提交**

```bash
git add -A
git commit -m "chore: 清理所有 BinaryPath/ParamsSchema 残留引用"
```

---

## Task 8: 更新 CLAUDE.md 和 docs/guide.md

**Files:**
- Modify: `CLAUDE.md`
- Modify: `docs/guide.md`

- [ ] **Step 1: 更新 CLAUDE.md 中的相关描述**

需要更新的部分：
1. Program 模型描述：BinaryPath → EntryFile，删除 ParamsSchema
2. buildArgs 函数说明：改为直接透传 []string
3. API 路由速查：程序注册改为 multipart/form-data
4. 工单提交 args 改为 []string
5. 代码结构中删除 buildArgs 引用

- [ ] **Step 2: 更新 docs/guide.md 中的 curl 示例**

程序注册示例从 JSON 改为 multipart curl。工单提交示例中 args 改为 []string。

- [ ] **Step 3: 提交**

```bash
git add CLAUDE.md docs/guide.md
git commit -m "docs: 更新文档适配文件上传程序注册"
```

---

## Task 9: 最终集成测试

**Files:** N/A (验证步骤)

- [ ] **Step 1: 编译后端**

Run: `go build -o loopguard ./cmd/loopguard`
Expected: 编译成功

- [ ] **Step 2: 运行全部后端测试**

Run: `go test -cover ./...`
Expected: 全部通过

- [ ] **Step 3: 前端构建**

Run: `cd web && pnpm build`
Expected: 构建成功

- [ ] **Step 4: 前端 lint**

Run: `cd web && pnpm lint`
Expected: 无错误
