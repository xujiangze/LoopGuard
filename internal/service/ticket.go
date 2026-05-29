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
		ApproverID: p.ApproverID,
	}
	if err := svc.store.CreateTicket(tk); err != nil {
		return nil, err
	}

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
