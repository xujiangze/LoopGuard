package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
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
		ApproverID: p.ApproverID,
	}
	if err := svc.store.CreateTicket(tk); err != nil {
		return nil, err
	}

	cliArgs := buildArgs(in.Args)
	res, runErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: p.BinaryPath, Interpreter: p.Interpreter, Args: cliArgs, DryRun: true,
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
}

func (svc *TicketService) Get(id uint64) (*model.Ticket, error) { return svc.store.GetTicket(id) }

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
	sb.WriteString(fmt.Sprintf("退出码: %d | %s", exitCode, result))
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

	var args map[string]any
	_ = json.Unmarshal(tk.Args, &args)
	res, runErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: p.BinaryPath, Interpreter: p.Interpreter, Args: buildArgs(args), DryRun: false,
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
