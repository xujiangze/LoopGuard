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

// WebhookTrigger 定义 webhook 触发接口
type WebhookTrigger interface {
	Trigger(ticketID uint64, eventType model.TicketStatus, approvalURL string)
}

type TicketService struct {
	store        *store.Store
	exec         executor.Executor
	workspaceDir string
	webhook      WebhookTrigger
	baseURL      string
}

func NewTicketService(s *store.Store, e executor.Executor, workspaceDir string) *TicketService {
	return &TicketService{store: s, exec: e, workspaceDir: workspaceDir}
}

func (svc *TicketService) SetWebhook(w WebhookTrigger, baseURL string) {
	svc.webhook = w
	svc.baseURL = baseURL
}

func (svc *TicketService) triggerWebhook(ticketID uint64, status model.TicketStatus) {
	if svc.webhook != nil {
		url := fmt.Sprintf("%s/tickets/%d", svc.baseURL, ticketID)
		svc.webhook.Trigger(ticketID, status, url)
	}
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

	workDir := filepath.Join(svc.workspaceDir, p.Project, p.Name)
	res, runErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: p.EntryFile, Interpreter: p.Interpreter, Args: in.Args, DryRun: true,
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
		svc.triggerWebhook(tk.ID, model.StatusDryrunFailed)
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
	svc.triggerWebhook(tk.ID, tk.Status)
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

	workDir := filepath.Join(svc.workspaceDir, p.Project, p.Name)
	res, runErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: p.EntryFile, Interpreter: p.Interpreter, Args: args, DryRun: false,
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
	svc.triggerWebhook(tk.ID, tk.Status)
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
	svc.triggerWebhook(tk.ID, model.StatusRejected)
	return tk, nil
}
