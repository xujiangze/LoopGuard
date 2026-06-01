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
	Interpreter  string
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

	help, err := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: in.BinaryPath, Interpreter: in.Interpreter, Args: []string{"--help"}, TimeoutSec: 10,
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
		Project: in.Project, Name: in.Name, BinaryPath: in.BinaryPath, Interpreter: in.Interpreter,
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
