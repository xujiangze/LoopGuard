# LoopGuard 后端 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 LoopGuard 后端——AI agent 危险操作的人工审批卡点服务，AI 经 API Key 提交工单，自动 `--only-print` dry-run 校验后由指定审批人放行并沙盒执行。

**Architecture:** 单体 Go/Gin 服务 + MySQL（GORM）。HTTP 层（API Key / JWT 中间件）→ Service 层（工单状态机、程序注册）→ Executor 接口（第一期 ProcessExecutor，os/exec + 进程组）。所有平台能力通过 CLI 子命令暴露。

**Tech Stack:** Go 1.26, Gin, GORM + MySQL 驱动, golang-jwt/jwt v5, bcrypt, cobra（CLI），testify + go-sqlmock（测试）。

---

## File Structure

```
cmd/loopguard/main.go              # cobra root + 子命令注册
internal/config/config.go          # 配置加载（env + 默认值）
internal/model/models.go           # GORM 模型：User/APIKey/Program/Ticket/Execution
internal/model/status.go           # 工单状态枚举 + 合法流转表
internal/store/store.go            # *gorm.DB 封装 + 各实体 CRUD
internal/auth/password.go          # bcrypt hash/verify
internal/auth/jwt.go               # JWT 签发/解析
internal/auth/apikey.go            # API Key 生成/哈希/校验
internal/executor/executor.go      # Executor 接口 + ExecRequest/ExecResult
internal/executor/process.go       # ProcessExecutor（os/exec + 进程组 + timeout）
internal/service/program.go        # 程序注册：抓 --help、探测 --only-print
internal/service/ticket.go         # 工单状态机：提交→dryrun→审批→执行
internal/service/dryrun.go         # dry-run 校验（DRYRUN-OK + 退出码 0）
internal/api/middleware.go         # APIKey / JWT 认证中间件
internal/api/router.go             # 路由注册
internal/api/ai_handler.go         # AI 接口 handler
internal/api/human_handler.go      # 人工审批 handler
internal/api/admin_handler.go      # 管理接口 handler
internal/cli/*.go                  # serve/migrate/admin/apikey 子命令
migrations/0001_init.sql           # 建表 SQL（参考，GORM AutoMigrate 为主）
```

每个文件单一职责；测试与被测文件同包，命名 `*_test.go`。

---

## Task 0: 项目骨架与依赖

**Files:**
- Create: `cmd/loopguard/main.go`
- Modify: `go.mod`

- [ ] **Step 1: 安装依赖**

Run:
```bash
go get github.com/gin-gonic/gin@latest && \
go get gorm.io/gorm@latest && \
go get gorm.io/driver/mysql@latest && \
go get github.com/golang-jwt/jwt/v5@latest && \
go get golang.org/x/crypto/bcrypt && \
go get github.com/spf13/cobra@latest && \
go get github.com/stretchr/testify@latest && \
go get github.com/DATA-DOG/go-sqlmock@latest
```
Expected: go.mod 更新，无报错。

- [ ] **Step 2: 写最小 main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "loopguard",
		Short: "LoopGuard - AI 危险操作人工审批卡点平台",
	}
	// 子命令在后续 Task 注册：root.AddCommand(...)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: 验证编译与 --help**

Run: `go build -o bin/loopguard ./cmd/loopguard && ./bin/loopguard --help`
Expected: 打印 "LoopGuard - AI 危险操作人工审批卡点平台" 及 usage。

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum cmd/loopguard/main.go
git commit -m "chore: 初始化 LoopGuard 后端骨架与依赖"
```

---

## Task 1: 配置加载

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: 写失败测试**

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("LOOPGUARD_DB_DSN", "")
	cfg := Load()
	assert.Equal(t, ":8080", cfg.HTTPAddr)
	assert.Equal(t, "process", cfg.ExecutorType)
	assert.Equal(t, "./workspace", cfg.WorkspaceDir)
	assert.NotEmpty(t, cfg.JWTSecret)
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("LOOPGUARD_HTTP_ADDR", ":9000")
	t.Setenv("LOOPGUARD_DB_DSN", "user:pass@tcp(localhost:3306)/lg")
	t.Setenv("LOOPGUARD_JWT_SECRET", "mysecret")
	cfg := Load()
	assert.Equal(t, ":9000", cfg.HTTPAddr)
	assert.Equal(t, "user:pass@tcp(localhost:3306)/lg", cfg.DBDSN)
	assert.Equal(t, "mysecret", cfg.JWTSecret)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/config/ -run TestLoad -v`
Expected: FAIL（config 包/Load 未定义）。

- [ ] **Step 3: 实现 config.go**

```go
package config

import "os"

type Config struct {
	HTTPAddr     string
	DBDSN        string
	JWTSecret    string
	ExecutorType string // process | docker
	WorkspaceDir string
	BaseURL      string // 用于拼 approval_url
}

func Load() Config {
	return Config{
		HTTPAddr:     env("LOOPGUARD_HTTP_ADDR", ":8080"),
		DBDSN:        env("LOOPGUARD_DB_DSN", ""),
		JWTSecret:    env("LOOPGUARD_JWT_SECRET", "dev-insecure-secret-change-me"),
		ExecutorType: env("LOOPGUARD_EXECUTOR_TYPE", "process"),
		WorkspaceDir: env("LOOPGUARD_WORKSPACE_DIR", "./workspace"),
		BaseURL:      env("LOOPGUARD_BASE_URL", "http://localhost:8080"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
```

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/config/ -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: 配置加载（env + 默认值）"
```

---

## Task 2: 工单状态枚举与流转表

**Files:**
- Create: `internal/model/status.go`
- Test: `internal/model/status_test.go`

- [ ] **Step 1: 写失败测试**

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanTransition(t *testing.T) {
	assert.True(t, CanTransition(StatusPendingDryrun, StatusPendingApproval))
	assert.True(t, CanTransition(StatusPendingDryrun, StatusDryrunFailed))
	assert.True(t, CanTransition(StatusPendingApproval, StatusApproved))
	assert.True(t, CanTransition(StatusPendingApproval, StatusRejected))
	assert.True(t, CanTransition(StatusApproved, StatusExecuting))
	assert.True(t, CanTransition(StatusExecuting, StatusDone))
	assert.True(t, CanTransition(StatusExecuting, StatusExecFailed))

	// 非法流转
	assert.False(t, CanTransition(StatusDone, StatusExecuting))
	assert.False(t, CanTransition(StatusRejected, StatusApproved))
	assert.False(t, CanTransition(StatusPendingDryrun, StatusDone))
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/model/ -run TestCanTransition -v`
Expected: FAIL（未定义）。

- [ ] **Step 3: 实现 status.go**

```go
package model

type TicketStatus string

const (
	StatusPendingDryrun   TicketStatus = "pending_dryrun"
	StatusDryrunFailed    TicketStatus = "dryrun_failed"
	StatusPendingApproval TicketStatus = "pending_approval"
	StatusApproved        TicketStatus = "approved"
	StatusExecuting       TicketStatus = "executing"
	StatusDone            TicketStatus = "done"
	StatusExecFailed      TicketStatus = "exec_failed"
	StatusRejected        TicketStatus = "rejected"
)

var allowed = map[TicketStatus][]TicketStatus{
	StatusPendingDryrun:   {StatusPendingApproval, StatusDryrunFailed},
	StatusPendingApproval: {StatusApproved, StatusRejected},
	StatusApproved:        {StatusExecuting},
	StatusExecuting:       {StatusDone, StatusExecFailed},
}

func CanTransition(from, to TicketStatus) bool {
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/model/ -run TestCanTransition -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/model/status.go internal/model/status_test.go
git commit -m "feat: 工单状态枚举与合法流转表"
```

---

## Task 3: GORM 数据模型

**Files:**
- Create: `internal/model/models.go`
- Test: `internal/model/models_test.go`

- [ ] **Step 1: 写失败测试（用 sqlite 内存库验证 AutoMigrate）**

先装 sqlite 驱动：`go get gorm.io/driver/sqlite@latest`

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&User{}, &APIKey{}, &Program{}, &Ticket{}, &Execution{})
	require.NoError(t, err)

	require.True(t, db.Migrator().HasTable(&User{}))
	require.True(t, db.Migrator().HasTable(&Ticket{}))
	require.True(t, db.Migrator().HasColumn(&Ticket{}, "dryrun_output"))
	require.True(t, db.Migrator().HasColumn(&Program{}, "supports_dryrun"))
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/model/ -run TestAutoMigrate -v`
Expected: FAIL（模型未定义）。

- [ ] **Step 3: 实现 models.go**

```go
package model

import (
	"time"

	"gorm.io/datatypes"
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
	ID             uint64         `gorm:"primaryKey" json:"id"`
	Project        string         `gorm:"size:128;not null;uniqueIndex:uk_project_name" json:"project"`
	Name           string         `gorm:"size:128;not null;uniqueIndex:uk_project_name" json:"name"`
	BinaryPath     string         `gorm:"size:512;not null" json:"binary_path"`
	HelpText       string         `gorm:"type:text" json:"help_text"`
	ParamsSchema   datatypes.JSON `gorm:"type:json" json:"params_schema"`
	ApproverID     uint64         `gorm:"not null" json:"approver_id"`
	TimeoutSec     int            `gorm:"not null;default:300" json:"timeout_sec"`
	SupportsDryrun bool           `gorm:"not null;default:true" json:"supports_dryrun"`
	Enabled        bool           `gorm:"not null;default:true" json:"enabled"`
	CreatedAt      time.Time      `json:"created_at"`
}

type Ticket struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	ProgramID    uint64         `gorm:"not null;index" json:"program_id"`
	Args         datatypes.JSON `gorm:"type:json;not null" json:"args"`
	Status       TicketStatus   `gorm:"type:varchar(32);not null;index" json:"status"`
	SubmittedBy  uint64         `gorm:"not null" json:"submitted_by"` // api_keys.id
	ApproverID   uint64         `gorm:"not null;index" json:"approver_id"`
	DryrunOutput string         `gorm:"type:mediumtext" json:"dryrun_output"`
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

需要 `go get gorm.io/datatypes@latest`。

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/model/ -run TestAutoMigrate -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/model/models.go internal/model/models_test.go go.mod go.sum
git commit -m "feat: GORM 数据模型（User/APIKey/Program/Ticket/Execution）"
```

---

## Task 4: 认证基础（bcrypt + JWT + API Key）

**Files:**
- Create: `internal/auth/password.go`, `internal/auth/jwt.go`, `internal/auth/apikey.go`
- Test: `internal/auth/auth_test.go`

- [ ] **Step 1: 写失败测试**

```go
package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordHashVerify(t *testing.T) {
	h, err := HashPassword("secret123")
	require.NoError(t, err)
	assert.NotEqual(t, "secret123", h)
	assert.True(t, VerifyPassword(h, "secret123"))
	assert.False(t, VerifyPassword(h, "wrong"))
}

func TestJWTSignParse(t *testing.T) {
	secret := "test-secret"
	tok, err := SignJWT(secret, 42, "admin")
	require.NoError(t, err)
	claims, err := ParseJWT(secret, tok)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), claims.UserID)
	assert.Equal(t, "admin", claims.Role)

	_, err = ParseJWT("wrong-secret", tok)
	assert.Error(t, err)
}

func TestAPIKeyGenerateHashVerify(t *testing.T) {
	plain := GenerateAPIKey()
	assert.True(t, len(plain) >= 32)
	h := HashAPIKey(plain)
	assert.Equal(t, h, HashAPIKey(plain)) // 确定性哈希便于查库
	assert.NotEqual(t, plain, h)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/auth/ -v`
Expected: FAIL（未定义）。

- [ ] **Step 3: 实现 password.go**

```go
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func VerifyPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
```

- [ ] **Step 4: 实现 jwt.go**

```go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint64 `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func SignJWT(secret string, userID uint64, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseJWT(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
```

- [ ] **Step 5: 实现 apikey.go**

```go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateAPIKey() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "lg_" + hex.EncodeToString(b)
}

// HashAPIKey 用 SHA-256 确定性哈希，便于按哈希直接查库。
func HashAPIKey(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 6: 运行验证通过**

Run: `go test ./internal/auth/ -v`
Expected: PASS（3 个测试全过）。

- [ ] **Step 7: Commit**

```bash
git add internal/auth/
git commit -m "feat: 认证基础（bcrypt 密码、JWT、API Key 生成与哈希）"
```

---

## Task 5: Store 数据访问层

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

- [ ] **Step 1: 写失败测试（sqlite 内存库）**

```go
package store

import (
	"testing"

	"LoopGuard/internal/model"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestStore(t *testing.T) *Store {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	s := New(db)
	require.NoError(t, s.AutoMigrate())
	return s
}

func TestCreateAndGetUser(t *testing.T) {
	s := newTestStore(t)
	u := &model.User{Username: "alice", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))
	require.NotZero(t, u.ID)

	got, err := s.GetUserByUsername("alice")
	require.NoError(t, err)
	require.Equal(t, model.RoleAdmin, got.Role)
}

func TestGetProgramByProjectName(t *testing.T) {
	s := newTestStore(t)
	u := &model.User{Username: "approver", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	p := &model.Program{Project: "demo", Name: "deploy", BinaryPath: "/bin/x", ApproverID: u.ID, Enabled: true}
	require.NoError(t, s.CreateProgram(p))

	got, err := s.GetProgramByProjectName("demo", "deploy")
	require.NoError(t, err)
	require.Equal(t, p.ID, got.ID)
}

func TestTicketLifecycle(t *testing.T) {
	s := newTestStore(t)
	tk := &model.Ticket{ProgramID: 1, ApproverID: 1, SubmittedBy: 1,
		Status: model.StatusPendingDryrun, Args: []byte(`{}`)}
	require.NoError(t, s.CreateTicket(tk))
	require.NotZero(t, tk.ID)

	tk.Status = model.StatusPendingApproval
	require.NoError(t, s.UpdateTicket(tk))

	got, err := s.GetTicket(tk.ID)
	require.NoError(t, err)
	require.Equal(t, model.StatusPendingApproval, got.Status)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/store/ -v`
Expected: FAIL（未定义）。

- [ ] **Step 3: 实现 store.go**

```go
package store

import (
	"LoopGuard/internal/model"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) DB() *gorm.DB { return s.db }

func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(
		&model.User{}, &model.APIKey{}, &model.Program{},
		&model.Ticket{}, &model.Execution{},
	)
}

// Users
func (s *Store) CreateUser(u *model.User) error { return s.db.Create(u).Error }
func (s *Store) GetUserByUsername(name string) (*model.User, error) {
	var u model.User
	err := s.db.Where("username = ?", name).First(&u).Error
	return &u, err
}
func (s *Store) GetUser(id uint64) (*model.User, error) {
	var u model.User
	err := s.db.First(&u, id).Error
	return &u, err
}

// API Keys
func (s *Store) CreateAPIKey(k *model.APIKey) error { return s.db.Create(k).Error }
func (s *Store) GetAPIKeyByHash(hash string) (*model.APIKey, error) {
	var k model.APIKey
	err := s.db.Where("key_hash = ? AND enabled = ?", hash, true).First(&k).Error
	return &k, err
}

// Programs
func (s *Store) CreateProgram(p *model.Program) error { return s.db.Create(p).Error }
func (s *Store) UpdateProgram(p *model.Program) error { return s.db.Save(p).Error }
func (s *Store) GetProgram(id uint64) (*model.Program, error) {
	var p model.Program
	err := s.db.First(&p, id).Error
	return &p, err
}
func (s *Store) GetProgramByProjectName(project, name string) (*model.Program, error) {
	var p model.Program
	err := s.db.Where("project = ? AND name = ?", project, name).First(&p).Error
	return &p, err
}
func (s *Store) ListPrograms() ([]model.Program, error) {
	var ps []model.Program
	err := s.db.Order("id desc").Find(&ps).Error
	return ps, err
}

// Tickets
func (s *Store) CreateTicket(t *model.Ticket) error { return s.db.Create(t).Error }
func (s *Store) UpdateTicket(t *model.Ticket) error { return s.db.Save(t).Error }
func (s *Store) GetTicket(id uint64) (*model.Ticket, error) {
	var t model.Ticket
	err := s.db.First(&t, id).Error
	return &t, err
}
func (s *Store) ListTicketsByApprover(approverID uint64, status model.TicketStatus) ([]model.Ticket, error) {
	q := s.db.Where("approver_id = ?", approverID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var ts []model.Ticket
	err := q.Order("id desc").Find(&ts).Error
	return ts, err
}

// Executions
func (s *Store) CreateExecution(e *model.Execution) error { return s.db.Create(e).Error }
func (s *Store) ListExecutionsByTicket(ticketID uint64) ([]model.Execution, error) {
	var es []model.Execution
	err := s.db.Where("ticket_id = ?", ticketID).Order("id asc").Find(&es).Error
	return es, err
}
```

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/store/ -v`
Expected: PASS（4 个测试全过）。

- [ ] **Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat: Store 数据访问层（User/APIKey/Program/Ticket/Execution CRUD）"
```

---

## Task 6: Executor 接口 + ProcessExecutor

**Files:**
- Create: `internal/executor/executor.go`, `internal/executor/process.go`
- Test: `internal/executor/process_test.go`

- [ ] **Step 1: 写接口定义 executor.go**

```go
package executor

import "context"

type ExecRequest struct {
	BinaryPath string
	Args       []string // 已拼好的参数（DryRun 时由调用方追加 --only-print）
	DryRun     bool
	TimeoutSec int
	WorkDir    string
	Env        []string // 白名单环境变量
}

type ExecResult struct {
	Command    string
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMs int64
	TimedOut   bool
}

type Executor interface {
	Run(ctx context.Context, req ExecRequest) (*ExecResult, error)
}
```

- [ ] **Step 2: 写失败测试 process_test.go**

```go
package executor

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessExecutorEcho(t *testing.T) {
	wd, _ := os.MkdirTemp("", "lg-test-*")
	defer os.RemoveAll(wd)
	e := NewProcessExecutor()
	res, err := e.Run(context.Background(), ExecRequest{
		BinaryPath: "/bin/echo",
		Args:       []string{"hello"},
		TimeoutSec: 5,
		WorkDir:    wd,
		Env:        []string{"PATH=/usr/bin:/bin"},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode)
	assert.Contains(t, res.Stdout, "hello")
	assert.False(t, res.TimedOut)
}

func TestProcessExecutorNonZeroExit(t *testing.T) {
	wd, _ := os.MkdirTemp("", "lg-test-*")
	defer os.RemoveAll(wd)
	e := NewProcessExecutor()
	res, err := e.Run(context.Background(), ExecRequest{
		BinaryPath: "/bin/sh",
		Args:       []string{"-c", "exit 3"},
		TimeoutSec: 5,
		WorkDir:    wd,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, res.ExitCode)
}

func TestProcessExecutorTimeout(t *testing.T) {
	wd, _ := os.MkdirTemp("", "lg-test-*")
	defer os.RemoveAll(wd)
	e := NewProcessExecutor()
	res, err := e.Run(context.Background(), ExecRequest{
		BinaryPath: "/bin/sh",
		Args:       []string{"-c", "sleep 10"},
		TimeoutSec: 1,
		WorkDir:    wd,
	})
	require.NoError(t, err)
	assert.True(t, res.TimedOut)
}
```

- [ ] **Step 3: 运行验证失败**

Run: `go test ./internal/executor/ -v`
Expected: FAIL（NewProcessExecutor 未定义）。

- [ ] **Step 4: 实现 process.go**

```go
package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type ProcessExecutor struct{}

func NewProcessExecutor() *ProcessExecutor { return &ProcessExecutor{} }

func (p *ProcessExecutor) Run(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	args := append([]string{}, req.Args...)
	if req.DryRun {
		args = append(args, "--only-print")
	}

	timeout := time.Duration(req.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, req.BinaryPath, args...)
	cmd.Dir = req.WorkDir
	cmd.Env = req.Env
	// 独立进程组，超时整组 kill 防残留子进程
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start).Milliseconds()

	res := &ExecResult{
		Command:    req.BinaryPath + " " + strings.Join(args, " "),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: dur,
	}

	if runCtx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		return res, nil
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, err // 程序无法启动（路径错等）才返回 error
	}
	res.ExitCode = 0
	return res, nil
}
```

- [ ] **Step 5: 运行验证通过**

Run: `go test ./internal/executor/ -v`
Expected: PASS（3 个测试全过）。

- [ ] **Step 6: Commit**

```bash
git add internal/executor/
git commit -m "feat: Executor 接口 + ProcessExecutor（os/exec + 进程组 + timeout）"
```

---

## Task 7: dry-run 校验

**Files:**
- Create: `internal/service/dryrun.go`
- Test: `internal/service/dryrun_test.go`

- [ ] **Step 1: 写失败测试**

```go
package service

import (
	"testing"

	"LoopGuard/internal/executor"

	"github.com/stretchr/testify/assert"
)

func TestValidateDryrun(t *testing.T) {
	// 通过：退出码 0 + 含 DRYRUN-OK
	ok := ValidateDryrun(&executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK\nwill delete x"})
	assert.True(t, ok.Passed)

	// 失败：退出码非 0
	r1 := ValidateDryrun(&executor.ExecResult{ExitCode: 1, Stdout: "DRYRUN-OK"})
	assert.False(t, r1.Passed)
	assert.Contains(t, r1.Reason, "退出码")

	// 失败：缺 DRYRUN-OK 标记
	r2 := ValidateDryrun(&executor.ExecResult{ExitCode: 0, Stdout: "did something"})
	assert.False(t, r2.Passed)
	assert.Contains(t, r2.Reason, "DRYRUN-OK")

	// 失败：超时
	r3 := ValidateDryrun(&executor.ExecResult{TimedOut: true})
	assert.False(t, r3.Passed)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/service/ -run TestValidateDryrun -v`
Expected: FAIL（未定义）。

- [ ] **Step 3: 实现 dryrun.go**

```go
package service

import (
	"strings"

	"LoopGuard/internal/executor"
)

const DryrunMarker = "DRYRUN-OK"

type DryrunResult struct {
	Passed bool
	Reason string
}

// ValidateDryrun 落实"信任+登记校验"：退出码必须为 0 且输出含 DRYRUN-OK 标记。
func ValidateDryrun(res *executor.ExecResult) DryrunResult {
	if res.TimedOut {
		return DryrunResult{false, "dry-run 执行超时"}
	}
	if res.ExitCode != 0 {
		return DryrunResult{false, "dry-run 退出码非 0（实际 " + itoa(res.ExitCode) + "）"}
	}
	if !strings.Contains(res.Stdout, DryrunMarker) {
		return DryrunResult{false, "dry-run 输出缺少 " + DryrunMarker + " 标记，疑似未正确实现 --only-print"}
	}
	return DryrunResult{true, ""}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
```

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/service/ -run TestValidateDryrun -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/service/dryrun.go internal/service/dryrun_test.go
git commit -m "feat: dry-run 校验（DRYRUN-OK 标记 + 退出码 0 双重校验）"
```

---

## Task 8: 程序注册 Service

**Files:**
- Create: `internal/service/program.go`
- Test: `internal/service/program_test.go`

- [ ] **Step 1: 写失败测试**

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

// fakeExecutor 返回预设结果，用于隔离测试 service 逻辑。
type fakeExecutor struct {
	result *executor.ExecResult
	err    error
}

func (f *fakeExecutor) Run(_ context.Context, _ executor.ExecRequest) (*executor.ExecResult, error) {
	return f.result, f.err
}

func newTestService(t *testing.T, exec executor.Executor) (*ProgramService, *store.Store) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return NewProgramService(s, exec), s
}

func TestRegisterProgramHelpProbe(t *testing.T) {
	// --help 探测返回退出码 0 且 --only-print 被识别（不报 unknown flag）
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s := newTestService(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", BinaryPath: "/bin/deploy",
		ApproverID: u.ID, TimeoutSec: 60, ParamsSchema: []byte(`{"env":"string"}`),
	})
	require.NoError(t, err)
	require.NotZero(t, p.ID)
	require.NotEmpty(t, p.HelpText)
	require.True(t, p.SupportsDryrun)
}

func TestRegisterProgramRejectsUnknownFlag(t *testing.T) {
	// --help 探测显示 unknown flag: --only-print → 拒绝注册
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 2, Stderr: "unknown flag: --only-print"}}
	svc, s := newTestService(t, fe)
	u := &model.User{Username: "appr2", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))

	_, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "bad", BinaryPath: "/bin/bad", ApproverID: u.ID,
	})
	require.Error(t, err)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/service/ -run TestRegisterProgram -v`
Expected: FAIL（未定义）。

- [ ] **Step 3: 实现 program.go**

```go
package service

import (
	"context"
	"errors"
	"strings"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"gorm.io/datatypes"
)

type ProgramService struct {
	store *store.Store
	exec  executor.Executor
}

func NewProgramService(s *store.Store, e executor.Executor) *ProgramService {
	return &ProgramService{store: s, exec: e}
}

type RegisterInput struct {
	Project      string
	Name         string
	BinaryPath   string
	ApproverID   uint64
	TimeoutSec   int
	ParamsSchema []byte
}

func (svc *ProgramService) Register(ctx context.Context, in RegisterInput) (*model.Program, error) {
	if in.Project == "" || in.Name == "" || in.BinaryPath == "" {
		return nil, errors.New("project/name/binary_path 必填")
	}
	if _, err := svc.store.GetUser(in.ApproverID); err != nil {
		return nil, errors.New("审批人不存在")
	}

	// 抓 --help 并探测 --only-print 是否被识别
	help, err := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: in.BinaryPath, Args: []string{"--help"}, TimeoutSec: 10,
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
		Project: in.Project, Name: in.Name, BinaryPath: in.BinaryPath,
		HelpText: combined, ParamsSchema: datatypes.JSON(in.ParamsSchema),
		ApproverID: in.ApproverID, TimeoutSec: timeout,
		SupportsDryrun: true, Enabled: true,
	}
	if err := svc.store.CreateProgram(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (svc *ProgramService) List() ([]model.Program, error) { return svc.store.ListPrograms() }
```

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/service/ -run TestRegisterProgram -v`
Expected: PASS（2 个测试全过）。

- [ ] **Step 5: Commit**

```bash
git add internal/service/program.go internal/service/program_test.go
git commit -m "feat: 程序注册 Service（抓 --help、探测 --only-print）"
```

---

## Task 9: 工单 Service（提交 + dry-run + 参数白名单）

**Files:**
- Create: `internal/service/ticket.go`
- Test: `internal/service/ticket_test.go`

- [ ] **Step 1: 写失败测试**

```go
package service

import (
	"context"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTicketService(t *testing.T, exec executor.Executor) (*TicketService, *store.Store) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return NewTicketService(s, exec), s
}

func seedProgram(t *testing.T, s *store.Store, schema string) *model.Program {
	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	p := &model.Program{Project: "demo", Name: "deploy", BinaryPath: "/bin/deploy",
		ApproverID: u.ID, TimeoutSec: 30, SupportsDryrun: true, Enabled: true,
		ParamsSchema: []byte(schema)}
	require.NoError(t, s.CreateProgram(p))
	return p
}

func TestSubmitTicketDryrunPass(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK\nwill deploy"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"env": "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "DRYRUN-OK")
}

func TestSubmitTicketDryrunFail(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "no marker"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7, Args: map[string]any{"env": "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDryrunFailed, tk.Status)
}

func TestSubmitRejectsUnknownArg(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	_, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"env": "prod", "force": "true"}, // force 不在白名单
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force")
}

func TestSubmitRejectsOnlyPrintInArgs(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"only-print":"bool"}`)

	_, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"only-print": "false"},
	})
	require.Error(t, err) // only-print 是保留字，禁止 AI 传
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/service/ -run TestSubmit -v`
Expected: FAIL（未定义）。

- [ ] **Step 3: 实现 ticket.go**

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"gorm.io/datatypes"
)

type TicketService struct {
	store *store.Store
	exec  executor.Executor
}

func NewTicketService(s *store.Store, e executor.Executor) *TicketService {
	return &TicketService{store: s, exec: e}
}

type SubmitInput struct {
	Project  string
	Name     string
	APIKeyID uint64
	Args     map[string]any
}

func (svc *TicketService) Submit(ctx context.Context, in SubmitInput) (*model.Ticket, error) {
	p, err := svc.store.GetProgramByProjectName(in.Project, in.Name)
	if err != nil {
		return nil, errors.New("程序未注册：" + in.Project + "/" + in.Name)
	}
	if !p.Enabled {
		return nil, errors.New("程序已禁用")
	}
	if err := validateArgs(p.ParamsSchema, in.Args); err != nil {
		return nil, err
	}

	argsJSON, _ := json.Marshal(in.Args)
	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(argsJSON),
		Status: model.StatusPendingDryrun, SubmittedBy: in.APIKeyID,
		ApproverID: p.ApproverID, // 快照
	}
	if err := svc.store.CreateTicket(tk); err != nil {
		return nil, err
	}

	// 自动 dry-run
	cliArgs := buildArgs(in.Args)
	res, runErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: p.BinaryPath, Args: cliArgs, DryRun: true,
		TimeoutSec: p.TimeoutSec, WorkDir: ".",
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
		tk.Status = model.StatusDryrunFailed
		tk.DryrunOutput = "dry-run 执行错误：" + errStr(runErr)
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
}

func (svc *TicketService) Get(id uint64) (*model.Ticket, error) { return svc.store.GetTicket(id) }

// validateArgs：args 的 key 必须全部在 schema 白名单内，且不得包含保留字 only-print。
func validateArgs(schema datatypes.JSON, args map[string]any) error {
	allowed := map[string]bool{}
	if len(schema) > 0 {
		m := map[string]any{}
		if err := json.Unmarshal(schema, &m); err == nil {
			for k := range m {
				allowed[k] = true
			}
		}
	}
	for k := range args {
		if k == "only-print" || k == "--only-print" {
			return errors.New("参数 only-print 为系统保留字，禁止传入")
		}
		if len(allowed) > 0 && !allowed[k] {
			return fmt.Errorf("参数 %s 不在程序白名单内", k)
		}
	}
	return nil
}

// buildArgs：把 map 转成 --key value 形式，key 排序保证确定性。
func buildArgs(args map[string]any) []string {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		out = append(out, "--"+k, fmt.Sprintf("%v", args[k]))
	}
	return out
}

func errStr(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
```

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/service/ -run TestSubmit -v`
Expected: PASS（4 个测试全过）。

- [ ] **Step 5: Commit**

```bash
git add internal/service/ticket.go internal/service/ticket_test.go
git commit -m "feat: 工单提交 Service（参数白名单 + 自动 dry-run + 状态流转）"
```

---

## Task 10: 工单审批与执行 Service

**Files:**
- Modify: `internal/service/ticket.go`（追加 Approve/Reject/execute）
- Test: `internal/service/ticket_approve_test.go`

- [ ] **Step 1: 写失败测试**

```go
package service

import (
	"context"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApproveTriggersExecution(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"env": "prod"}})
	require.NoError(t, err)
	require.Equal(t, model.StatusPendingApproval, tk.Status)

	// 切换 fake 为正式执行结果
	fe.result = &executor.ExecResult{ExitCode: 0, Stdout: "deployed"}
	out, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusDone, out.Status)
	assert.NotNil(t, out.ApprovedBy)
	assert.Equal(t, p.ApproverID, *out.ApprovedBy)
}

func TestApproveWrongUserRejected(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)
	tk, _ := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"env": "prod"}})

	_, err := svc.Approve(context.Background(), tk.ID, p.ApproverID+999) // 非指定审批人
	require.Error(t, err)
}

func TestApproveExecFailed(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)
	tk, _ := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"env": "prod"}})

	fe.result = &executor.ExecResult{ExitCode: 5, Stderr: "boom"}
	out, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusExecFailed, out.Status)
}

func TestReject(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)
	tk, _ := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"env": "prod"}})

	out, err := svc.Reject(tk.ID, p.ApproverID, "太危险")
	require.NoError(t, err)
	assert.Equal(t, model.StatusRejected, out.Status)
	assert.Equal(t, "太危险", out.RejectReason)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/service/ -run "TestApprove|TestReject" -v`
Expected: FAIL（Approve/Reject 未定义）。

- [ ] **Step 3: 在 ticket.go 追加实现**

```go
// Approve 校验审批人身份 → 流转到 approved/executing → 去掉 --only-print 执行。
func (svc *TicketService) Approve(ctx context.Context, ticketID, userID uint64) (*model.Ticket, error) {
	tk, err := svc.store.GetTicket(ticketID)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	if tk.ApproverID != userID {
		// admin 也可审批的逻辑放在 handler 层判断后传入有效 userID
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

	var args map[string]any
	_ = json.Unmarshal(tk.Args, &args)
	res, runErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: p.BinaryPath, Args: buildArgs(args), DryRun: false,
		TimeoutSec: p.TimeoutSec, WorkDir: ".",
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

	if runErr != nil || res == nil || res.ExitCode != 0 || res.TimedOut {
		tk.Status = model.StatusExecFailed
	} else {
		tk.Status = model.StatusDone
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

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/service/ -run "TestApprove|TestReject" -v`
Expected: PASS（4 个测试全过）。

- [ ] **Step 5: Commit**

```bash
git add internal/service/ticket.go internal/service/ticket_approve_test.go
git commit -m "feat: 工单审批与执行 Service（审批人校验 + 去 --only-print 执行）"
```

---

## Task 11: 认证中间件

**Files:**
- Create: `internal/api/middleware.go`
- Test: `internal/api/middleware_test.go`

- [ ] **Step 1: 写失败测试**

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testStore(t *testing.T) *store.Store {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return s
}

func TestAPIKeyMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	plain := auth.GenerateAPIKey()
	require.NoError(t, s.CreateAPIKey(&model.APIKey{Name: "ai", KeyHash: auth.HashAPIKey(plain), Enabled: true}))

	r := gin.New()
	r.GET("/x", APIKeyAuth(s), func(c *gin.Context) { c.String(200, "ok") })

	// 有效 key
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-API-Key", plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// 无 key
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestJWTMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "s"
	tok, _ := auth.SignJWT(secret, 5, "admin")

	r := gin.New()
	r.GET("/y", JWTAuth(secret), func(c *gin.Context) {
		uid := c.GetUint64("user_id")
		c.JSON(200, gin.H{"uid": uid})
	})
	req := httptest.NewRequest("GET", "/y", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "5")
}

func TestAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/z", func(c *gin.Context) { c.Set("role", "user"); c.Next() }, AdminOnly(), func(c *gin.Context) { c.String(200, "ok") })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/z", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/api/ -run "Middleware|AdminOnly" -v`
Expected: FAIL（未定义）。

- [ ] **Step 3: 实现 middleware.go**

```go
package api

import (
	"net/http"
	"strings"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		plain := c.GetHeader("X-API-Key")
		if plain == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 X-API-Key"})
			return
		}
		k, err := s.GetAPIKeyByHash(auth.HashAPIKey(plain))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效 API Key"})
			return
		}
		c.Set("api_key_id", k.ID)
		c.Next()
	}
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 Bearer token"})
			return
		}
		claims, err := auth.ParseJWT(secret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效 token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("role") != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/api/ -run "Middleware|AdminOnly" -v`
Expected: PASS（3 个测试全过）。

- [ ] **Step 5: Commit**

```bash
git add internal/api/middleware.go internal/api/middleware_test.go
git commit -m "feat: 认证中间件（API Key / JWT / AdminOnly）"
```

---

## Task 12: AI 接口 Handler（提交 + 轮询）

**Files:**
- Create: `internal/api/ai_handler.go`
- Test: `internal/api/ai_handler_test.go`

- [ ] **Step 1: 写失败测试**

```go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/config"
	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExec struct{ res *executor.ExecResult }

func (f *fakeExec) Run(_ interface{ Done() <-chan struct{} }, _ executor.ExecRequest) (*executor.ExecResult, error) {
	return f.res, nil
}

func setupAI(t *testing.T) (*gin.Engine, *store.Store) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	// 审批人 + 程序
	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	require.NoError(t, s.CreateProgram(&model.Program{Project: "demo", Name: "deploy",
		BinaryPath: "/bin/echo", ApproverID: u.ID, TimeoutSec: 10, SupportsDryrun: true,
		Enabled: true, ParamsSchema: []byte(`{"msg":"string"}`)}))

	// 真实 ProcessExecutor + echo "DRYRUN-OK"
	ex := executor.NewProcessExecutor()
	ticketSvc := service.NewTicketService(s, ex)
	cfg := config.Config{BaseURL: "http://test"}
	h := NewAIHandler(ticketSvc, cfg)

	r := gin.New()
	r.POST("/api/v1/tickets", func(c *gin.Context) { c.Set("api_key_id", uint64(1)); c.Next() }, h.Submit)
	r.GET("/api/v1/tickets/:id", h.Get)
	return r, s
}

func TestAISubmitReturnsApprovalURL(t *testing.T) {
	// 用 /bin/echo 当程序：echo --only-print --msg DRYRUN-OK 会打印含 DRYRUN-OK 的行
	r, s := setupAI(t)
	_ = s
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
}
```

> 注：测试用 `/bin/echo` 作为被托管程序，`buildArgs` 会拼出 `--msg DRYRUN-OK`，加 `--only-print` 后 echo 全部原样打印，stdout 含 `DRYRUN-OK` 且退出码 0，正好通过 dry-run 校验。`fakeExec` 类型不用则删除。

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/api/ -run TestAISubmit -v`
Expected: FAIL（NewAIHandler 未定义）。

- [ ] **Step 3: 实现 ai_handler.go**

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
	Project string         `json:"project" binding:"required"`
	Name    string         `json:"name" binding:"required"`
	Args    map[string]any `json:"args"`
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

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/api/ -run TestAISubmit -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/api/ai_handler.go internal/api/ai_handler_test.go
git commit -m "feat: AI 接口 Handler（提交工单返回 approval_url + next_action 指引）"
```

---

## Task 13: 人工审批 Handler（登录 + 列表 + 审批/驳回）

**Files:**
- Create: `internal/api/human_handler.go`
- Test: `internal/api/human_handler_test.go`

- [ ] **Step 1: 写失败测试**

```go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/config"
	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginAndApprove(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	pw, _ := auth.HashPassword("pw123")
	appr := &model.User{Username: "appr", PasswordHash: pw, Role: model.RoleUser}
	require.NoError(t, s.CreateUser(appr))
	require.NoError(t, s.CreateProgram(&model.Program{Project: "demo", Name: "deploy",
		BinaryPath: "/bin/echo", ApproverID: appr.ID, TimeoutSec: 10, SupportsDryrun: true,
		Enabled: true, ParamsSchema: []byte(`{"msg":"string"}`)}))

	ex := executor.NewProcessExecutor()
	ticketSvc := service.NewTicketService(s, ex)
	tk, err := ticketSvc.Submit(nil, service.SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"msg": "DRYRUN-OK"}})
	require.NoError(t, err)
	require.Equal(t, model.StatusPendingApproval, tk.Status)

	cfg := config.Config{JWTSecret: "s"}
	h := NewHumanHandler(s, ticketSvc, cfg)
	r := gin.New()
	r.POST("/login", h.Login)
	auth := r.Group("/", JWTAuth(cfg.JWTSecret))
	auth.POST("/tickets/:id/approve", h.Approve)

	// 登录
	body, _ := json.Marshal(map[string]string{"username": "appr", "password": "pw123"})
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, mustReq("POST", "/login", body))
	require.Equal(t, http.StatusOK, lw.Code)
	var lr map[string]string
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &lr))
	token := lr["token"]
	require.NotEmpty(t, token)

	// 审批
	aw := httptest.NewRecorder()
	areq := mustReq("POST", "/tickets/1/approve", nil)
	areq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(aw, areq)
	assert.Equal(t, http.StatusOK, aw.Code)

	got, _ := s.GetTicket(tk.ID)
	assert.Equal(t, model.StatusDone, got.Status)
}

func mustReq(method, path string, body []byte) *http.Request {
	if body == nil {
		return httptest.NewRequest(method, path, nil)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/api/ -run TestLoginAndApprove -v`
Expected: FAIL（NewHumanHandler 未定义）。

- [ ] **Step 3: 实现 human_handler.go**

```go
package api

import (
	"net/http"
	"strconv"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/config"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
)

type HumanHandler struct {
	store   *store.Store
	tickets *service.TicketService
	cfg     config.Config
}

func NewHumanHandler(s *store.Store, t *service.TicketService, cfg config.Config) *HumanHandler {
	return &HumanHandler{store: s, tickets: t, cfg: cfg}
}

func (h *HumanHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.store.GetUserByUsername(req.Username)
	if err != nil || !auth.VerifyPassword(u.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	tok, err := auth.SignJWT(h.cfg.JWTSecret, u.ID, string(u.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签发 token 失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "role": u.Role, "user_id": u.ID})
}

// effectiveApprover：admin 可代任意工单审批，否则必须是工单指定审批人。
func (h *HumanHandler) effectiveApprover(c *gin.Context, tk *model.Ticket) uint64 {
	if c.GetString("role") == "admin" {
		return tk.ApproverID // 以工单指定人身份通过校验
	}
	return c.GetUint64("user_id")
}

func (h *HumanHandler) Approve(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	tk, err := h.store.GetTicket(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	out, err := h.tickets.Approve(c.Request.Context(), id, h.effectiveApprover(c, tk))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *HumanHandler) Reject(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	tk, err := h.store.GetTicket(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	out, err := h.tickets.Reject(id, h.effectiveApprover(c, tk), req.Reason)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *HumanHandler) ListMine(c *gin.Context) {
	uid := c.GetUint64("user_id")
	status := model.TicketStatus(c.Query("status"))
	ts, err := h.store.ListTicketsByApprover(uid, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ts)
}
```

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/api/ -run TestLoginAndApprove -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/api/human_handler.go internal/api/human_handler_test.go
git commit -m "feat: 人工审批 Handler（登录 + 待审批列表 + 审批/驳回，含审批人校验）"
```

---

## Task 14: 管理 Handler（注册程序 + 建用户 + 建 API Key）

**Files:**
- Create: `internal/api/admin_handler.go`
- Test: `internal/api/admin_handler_test.go`

- [ ] **Step 1: 写失败测试**

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminCreateAPIKeyReturnsPlainOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	ex := executor.NewProcessExecutor()
	progSvc := service.NewProgramService(s, ex)
	h := NewAdminHandler(s, progSvc)

	r := gin.New()
	r.POST("/api-keys", h.CreateAPIKey)
	body := []byte(`{"name":"hermes-agent"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, mustReq("POST", "/api-keys", body))

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["api_key"]) // 明文只此一次返回
}

func TestAdminCreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	h := NewAdminHandler(s, service.NewProgramService(s, executor.NewProcessExecutor()))
	r := gin.New()
	r.POST("/users", h.CreateUser)
	body := []byte(`{"username":"bob","password":"pw123456","role":"user"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, mustReq("POST", "/users", body))
	require.Equal(t, http.StatusOK, w.Code)

	u, err := s.GetUserByUsername("bob")
	require.NoError(t, err)
	assert.Equal(t, model.RoleUser, u.Role)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/api/ -run TestAdmin -v`
Expected: FAIL（NewAdminHandler 未定义）。

- [ ] **Step 3: 实现 admin_handler.go**

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
	store    *store.Store
	programs *service.ProgramService
}

func NewAdminHandler(s *store.Store, p *service.ProgramService) *AdminHandler {
	return &AdminHandler{store: s, programs: p}
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
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
	// 明文只返回这一次
	c.JSON(http.StatusOK, gin.H{"id": k.ID, "name": k.Name, "api_key": plain})
}

func (h *AdminHandler) CreateProgram(c *gin.Context) {
	var req struct {
		Project      string          `json:"project" binding:"required"`
		Name         string          `json:"name" binding:"required"`
		BinaryPath   string          `json:"binary_path" binding:"required"`
		ApproverID   uint64          `json:"approver_id" binding:"required"`
		TimeoutSec   int             `json:"timeout_sec"`
		ParamsSchema json.RawMessage `json:"params_schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.programs.Register(c.Request.Context(), service.RegisterInput{
		Project: req.Project, Name: req.Name, BinaryPath: req.BinaryPath,
		ApproverID: req.ApproverID, TimeoutSec: req.TimeoutSec, ParamsSchema: req.ParamsSchema,
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
	p, err := h.store.GetProgram(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	var req struct {
		Enabled    *bool   `json:"enabled"`
		ApproverID *uint64 `json:"approver_id"`
		TimeoutSec *int    `json:"timeout_sec"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	if req.ApproverID != nil {
		p.ApproverID = *req.ApproverID
	}
	if req.TimeoutSec != nil {
		p.TimeoutSec = *req.TimeoutSec
	}
	if err := h.store.UpdateProgram(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}
```

需在文件顶部 import 块加入 `"encoding/json"`。

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/api/ -run TestAdmin -v`
Expected: PASS（2 个测试全过）。

- [ ] **Step 5: Commit**

```bash
git add internal/api/admin_handler.go internal/api/admin_handler_test.go
git commit -m "feat: 管理 Handler（注册程序 / 建用户 / 建 API Key 明文返回一次）"
```

---

## Task 15: 路由组装

**Files:**
- Create: `internal/api/router.go`
- Test: `internal/api/router_test.go`

- [ ] **Step 1: 写失败测试**

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/config"
	"LoopGuard/internal/executor"
	"LoopGuard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRouterWiring(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	ex := executor.NewProcessExecutor()
	deps := Deps{
		Store:      s,
		TicketSvc:  service.NewTicketService(s, ex),
		ProgramSvc: service.NewProgramService(s, ex),
		Cfg:        config.Config{JWTSecret: "s", BaseURL: "http://t"},
	}
	r := NewRouter(deps)

	// 未带 API Key 的提交应 401
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/tickets", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 未带 JWT 的审批列表应 401
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/api/v1/tickets", nil))
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/api/ -run TestRouterWiring -v`
Expected: FAIL（NewRouter/Deps 未定义）。

- [ ] **Step 3: 实现 router.go**

```go
package api

import (
	"LoopGuard/internal/config"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

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

	ai := NewAIHandler(d.TicketSvc, d.Cfg)
	human := NewHumanHandler(d.Store, d.TicketSvc, d.Cfg)
	admin := NewAdminHandler(d.Store, d.ProgramSvc)

	v1 := r.Group("/api/v1")

	// 公共：登录
	v1.POST("/auth/login", human.Login)

	// AI 接口（X-API-Key）
	aiGrp := v1.Group("", APIKeyAuth(d.Store))
	aiGrp.POST("/tickets", ai.Submit)
	aiGrp.GET("/tickets/:id", ai.Get)

	// 人工接口（JWT）
	jwtGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret))
	jwtGrp.GET("/tickets", human.ListMine)
	jwtGrp.POST("/tickets/:id/approve", human.Approve)
	jwtGrp.POST("/tickets/:id/reject", human.Reject)

	// 管理接口（JWT + admin）
	adminGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret), AdminOnly())
	adminGrp.POST("/programs", admin.CreateProgram)
	adminGrp.GET("/programs", admin.ListPrograms)
	adminGrp.PUT("/programs/:id", admin.UpdateProgram)
	adminGrp.POST("/users", admin.CreateUser)
	adminGrp.POST("/api-keys", admin.CreateAPIKey)

	return r
}
```

> 注意：`GET /api/v1/tickets/:id`（AI 轮询，API Key）与 `GET /api/v1/tickets`（人工列表，JWT）路径不冲突；`:id` 仅匹配单段。AI 的 `GET /tickets/:id` 在 aiGrp，人工无单条 GET——审批页通过 `GET /tickets` 列表 + 详情字段获取。若前端需 JWT 下查单条详情，在 jwtGrp 另加 `GET /tickets/:id/detail` 避免与 AI 的 API Key 路由方法冲突。

- [ ] **Step 4: 运行验证通过**

Run: `go test ./internal/api/ -run TestRouterWiring -v`
Expected: PASS。

- [ ] **Step 5: 运行全部 api 测试确认无回归**

Run: `go test ./internal/api/ -v`
Expected: 全部 PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/api/router.go internal/api/router_test.go
git commit -m "feat: 路由组装（AI/人工/管理三组中间件）"
```

---

## Task 16: CLI 子命令（serve / migrate / admin create-user / apikey create）

**Files:**
- Create: `internal/cli/serve.go`, `internal/cli/migrate.go`, `internal/cli/admin.go`, `internal/cli/apikey.go`, `internal/cli/db.go`
- Modify: `cmd/loopguard/main.go`
- Test: `internal/cli/admin_test.go`

- [ ] **Step 1: 写失败测试（验证 admin create-user 真的建库写人）**

```go
package cli

import (
	"testing"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateUserLogic(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())

	err := CreateUserInStore(s, "root", "rootpw123", true)
	require.NoError(t, err)

	u, err := s.GetUserByUsername("root")
	require.NoError(t, err)
	assert.Equal(t, "admin", string(u.Role))
	assert.True(t, auth.VerifyPassword(u.PasswordHash, "rootpw123"))
}
```

- [ ] **Step 2: 运行验证失败**

Run: `go test ./internal/cli/ -run TestCreateUserLogic -v`
Expected: FAIL（未定义）。

- [ ] **Step 3: 实现 db.go（DB 连接工具）**

```go
package cli

import (
	"LoopGuard/internal/config"
	"LoopGuard/internal/store"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func openStore(cfg config.Config) (*store.Store, error) {
	db, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return store.New(db), nil
}
```

- [ ] **Step 4: 实现 admin.go（含可测纯逻辑 + cobra 命令）**

```go
package cli

import (
	"fmt"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/config"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/spf13/cobra"
)

// CreateUserInStore 纯逻辑，便于测试。
func CreateUserInStore(s *store.Store, username, password string, admin bool) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	role := model.RoleUser
	if admin {
		role = model.RoleAdmin
	}
	return s.CreateUser(&model.User{Username: username, PasswordHash: hash, Role: role})
}

func AdminCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "admin", Short: "用户管理"}
	var username, password string
	var isAdmin bool
	create := &cobra.Command{
		Use:   "create-user",
		Short: "创建用户（首个 admin 用 --admin）",
		RunE: func(c *cobra.Command, _ []string) error {
			s, err := openStore(config.Load())
			if err != nil {
				return err
			}
			if err := CreateUserInStore(s, username, password, isAdmin); err != nil {
				return err
			}
			fmt.Printf("已创建用户 %s（admin=%v）\n", username, isAdmin)
			return nil
		},
	}
	create.Flags().StringVar(&username, "username", "", "用户名")
	create.Flags().StringVar(&password, "password", "", "密码")
	create.Flags().BoolVar(&isAdmin, "admin", false, "设为管理员")
	_ = create.MarkFlagRequired("username")
	_ = create.MarkFlagRequired("password")
	cmd.AddCommand(create)
	return cmd
}
```

- [ ] **Step 5: 实现 migrate.go**

```go
package cli

import (
	"fmt"

	"LoopGuard/internal/config"

	"github.com/spf13/cobra"
)

func MigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "执行数据库迁移（AutoMigrate）",
		RunE: func(c *cobra.Command, _ []string) error {
			s, err := openStore(config.Load())
			if err != nil {
				return err
			}
			if err := s.AutoMigrate(); err != nil {
				return err
			}
			fmt.Println("迁移完成")
			return nil
		},
	}
}
```

- [ ] **Step 6: 实现 apikey.go**

```go
package cli

import (
	"fmt"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/config"
	"LoopGuard/internal/model"

	"github.com/spf13/cobra"
)

func APIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "apikey", Short: "AI 服务账号 Key 管理"}
	var name string
	create := &cobra.Command{
		Use:   "create",
		Short: "创建 API Key（明文仅打印一次）",
		RunE: func(c *cobra.Command, _ []string) error {
			s, err := openStore(config.Load())
			if err != nil {
				return err
			}
			plain := auth.GenerateAPIKey()
			k := &model.APIKey{Name: name, KeyHash: auth.HashAPIKey(plain), Enabled: true}
			if err := s.CreateAPIKey(k); err != nil {
				return err
			}
			fmt.Printf("API Key 已创建（请妥善保存，只显示这一次）：\n  name: %s\n  key:  %s\n", name, plain)
			return nil
		},
	}
	create.Flags().StringVar(&name, "name", "", "Key 名称，如 hermes-agent")
	_ = create.MarkFlagRequired("name")
	cmd.AddCommand(create)
	return cmd
}
```

- [ ] **Step 7: 实现 serve.go**

```go
package cli

import (
	"fmt"

	"LoopGuard/internal/api"
	"LoopGuard/internal/config"
	"LoopGuard/internal/executor"
	"LoopGuard/internal/service"

	"github.com/spf13/cobra"
)

func ServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "启动 HTTP 服务",
		RunE: func(c *cobra.Command, _ []string) error {
			cfg := config.Load()
			s, err := openStore(cfg)
			if err != nil {
				return err
			}
			if err := s.AutoMigrate(); err != nil {
				return err
			}
			var ex executor.Executor = executor.NewProcessExecutor()
			deps := api.Deps{
				Store:      s,
				TicketSvc:  service.NewTicketService(s, ex),
				ProgramSvc: service.NewProgramService(s, ex),
				Cfg:        cfg,
			}
			r := api.NewRouter(deps)
			fmt.Printf("LoopGuard 监听 %s\n", cfg.HTTPAddr)
			return r.Run(cfg.HTTPAddr)
		},
	}
}
```

- [ ] **Step 8: 改 main.go 注册子命令**

```go
package main

import (
	"fmt"
	"os"

	"LoopGuard/internal/cli"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "loopguard",
		Short: "LoopGuard - AI 危险操作人工审批卡点平台",
	}
	root.AddCommand(cli.ServeCmd(), cli.MigrateCmd(), cli.AdminCmd(), cli.APIKeyCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 9: 运行验证通过 + 编译 + --help**

Run:
```bash
go test ./internal/cli/ -v && \
go build -o bin/loopguard ./cmd/loopguard && \
./bin/loopguard --help && ./bin/loopguard admin --help
```
Expected: 测试 PASS；`--help` 列出 serve/migrate/admin/apikey；`admin --help` 显示 create-user。

- [ ] **Step 10: Commit**

```bash
git add internal/cli/ cmd/loopguard/main.go
git commit -m "feat: CLI 子命令（serve/migrate/admin create-user/apikey create）"
```

---

## Task 17: 端到端冒烟（全量测试 + go vet）

**Files:**
- 无新增，验证整体。

- [ ] **Step 1: 全量测试**

Run: `go test ./... -v`
Expected: 所有包 PASS。

- [ ] **Step 2: go vet**

Run: `go vet ./...`
Expected: 无输出（无问题）。

- [ ] **Step 3: 编译产物 --help 自描述检查**

Run: `go build -o bin/loopguard ./cmd/loopguard && ./bin/loopguard --help`
Expected: 列出全部子命令，符合"二进制 --help 可获取所有能力"要求。

- [ ] **Step 4: Commit（如有格式化改动）**

```bash
gofmt -w ./internal ./cmd && git add -A && git commit -m "chore: 后端端到端冒烟通过，gofmt 整理" || echo "无改动"
```

---

## Self-Review 结果

**Spec 覆盖检查（逐节对照设计文档）：**

- §4 工单状态机 → Task 2（流转表）+ Task 9/10（实际流转）✅
- §5 数据模型 5 张表 → Task 3 ✅
- §6.1 AI 接口（提交+轮询+next_action）→ Task 12 ✅
- §6.2 人工接口（登录/列表/审批/驳回 + 审批人校验）→ Task 13 ✅
- §6.3 管理接口（注册程序/建用户/建 Key）→ Task 14 ✅
- §7.1 Executor 接口 → Task 6 ✅
- §7.2 ProcessExecutor（独立目录/环境白名单/超时/退出码）→ Task 6 ✅
- §7.3 --only-print 三道关：注册探测 → Task 8；dry-run 双重校验 → Task 7+9；执行去 flag → Task 10 ✅
- §8 用户系统两角色 + 首个 admin → Task 4 + Task 16 ✅
- §9 项目结构 → File Structure 节 ✅
- §10 CLI 子命令 → Task 16 ✅
- §11 前端 → **不在本计划**（设计已说明前端等后端完成后单独出计划）

**类型一致性检查：**
- `TicketStatus` 常量在 Task 2 定义，Task 9/10/12/13 一致引用 ✅
- `Executor` 接口签名 `Run(ctx, ExecRequest)` 在 Task 6 定义，service/handler 测试中 `fakeExecutor` 匹配 ✅（注：Task 12 测试里的 `fakeExec` 签名错误，已在该任务注释中说明删除，实际用真实 ProcessExecutor + /bin/echo）
- `buildArgs` / `ValidateDryrun` / `DryrunMarker` 命名跨 Task 7/9/10 一致 ✅
- `store` 方法名（CreateTicket/UpdateTicket/GetTicket/ListTicketsByApprover 等）Task 5 定义，后续一致 ✅

**Placeholder 扫描：** 无 TBD/TODO，每个代码步骤含完整可编译代码 ✅

**已知简化（第一期 YAGNI，符合设计非目标）：**
- 执行为同步阻塞（审批请求里直接跑），任务量上来再引入 Worker
- 环境变量白名单第一期在 service 层用固定 PATH，程序级白名单留待需要时扩展
