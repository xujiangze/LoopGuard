package service

import (
	"context"
	"errors"
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
