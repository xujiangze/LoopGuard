package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"
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
		SupportsDryrun: true, Enabled: true, CurrentVersion: 1,
	}
	if err := svc.store.CreateProgram(p); err != nil {
		return nil, err
	}

	// 创建 v1 快照
	if err := svc.createVersionSnapshot(p, combined, "system"); err != nil {
		return nil, fmt.Errorf("创建版本快照失败: %w", err)
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
		p.CurrentVersion++
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

	// 文件变更时创建版本快照（当前目录内容即为新版本内容）
	if len(in.Files) > 0 {
		if err := svc.createVersionSnapshot(p, p.HelpText, "system"); err != nil {
			return nil, fmt.Errorf("创建版本快照失败: %w", err)
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

type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsEntry bool   `json:"is_entry"`
	ModTime string `json:"mod_time"`
}

func (svc *ProgramService) ListVersions(programID uint64) ([]model.ProgramVersion, error) {
	return svc.store.ListProgramVersions(programID)
}

func (svc *ProgramService) GetCurrentFiles(p *model.Program) ([]FileInfo, error) {
	dir := programDir(svc.workspaceDir, p.Project, p.Name)
	return listFilesInDir(dir, p.EntryFile)
}

func (svc *ProgramService) GetCurrentFileContent(p *model.Program, filename string) (string, error) {
	if err := sanitizeFilename(filename); err != nil {
		return "", err
	}
	path := filepath.Join(svc.workspaceDir, p.Project, p.Name, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("文件不存在")
	}
	return string(data), nil
}

func (svc *ProgramService) GetVersionFiles(programID uint64, version int) ([]FileInfo, error) {
	pv, err := svc.store.GetProgramVersion(programID, version)
	if err != nil {
		return nil, fmt.Errorf("版本不存在")
	}
	dir := filepath.Join(svc.workspaceDir, ".versions", fmt.Sprintf("%d", programID), fmt.Sprintf("v%d", version))
	return listFilesInDir(dir, pv.EntryFile)
}

func (svc *ProgramService) GetVersionFileContent(programID uint64, version int, filename string) (string, error) {
	if err := sanitizeFilename(filename); err != nil {
		return "", err
	}
	path := filepath.Join(svc.workspaceDir, ".versions", fmt.Sprintf("%d", programID), fmt.Sprintf("v%d", version), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("文件不存在")
	}
	return string(data), nil
}

func (svc *ProgramService) Rollback(ctx context.Context, programID uint64, targetVersion int, createdBy string) (*model.Program, error) {
	p, err := svc.store.GetProgram(programID)
	if err != nil {
		return nil, errors.New("程序不存在")
	}
	pv, err := svc.store.GetProgramVersion(programID, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("目标版本 %d 不存在", targetVersion)
	}

	// 用目标版本的快照内容覆盖当前目录
	currentDir := programDir(svc.workspaceDir, p.Project, p.Name)
	snapshotDir := filepath.Join(svc.workspaceDir, ".versions", fmt.Sprintf("%d", programID), fmt.Sprintf("v%d", targetVersion))
	if err := restoreFromSnapshot(snapshotDir, currentDir); err != nil {
		return nil, fmt.Errorf("恢复快照失败: %w", err)
	}

	// 版本号自增
	p.CurrentVersion++
	p.EntryFile = pv.EntryFile

	// 重新执行 --help
	binaryPath := filepath.Join(currentDir, p.EntryFile)
	help, execErr := svc.exec.Run(ctx, executor.ExecRequest{
		BinaryPath: binaryPath, Interpreter: pv.Interpreter, Args: []string{"--help"}, TimeoutSec: 10, WorkDir: currentDir,
	})
	helpText := ""
	if execErr == nil && help != nil {
		helpText = help.Stdout + "\n" + help.Stderr
		p.HelpText = helpText
	}

	// 创建新版本快照（标记 is_rollback=true）
	if err := svc.createVersionSnapshot(p, helpText, createdBy); err != nil {
		return nil, fmt.Errorf("创建版本快照失败: %w", err)
	}
	// 标记为回滚版本
	lastPv, _ := svc.store.GetProgramVersion(programID, int(p.CurrentVersion))
	if lastPv != nil {
		lastPv.IsRollback = true
		svc.store.DB().Model(lastPv).Update("is_rollback", true)
	}

	if err := svc.store.UpdateProgram(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (svc *ProgramService) DeleteProgram(id uint64) error {
	return svc.store.DeleteProgramWithCascade(id, svc.workspaceDir)
}

func listFilesInDir(dir, entryFile string) ([]FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}
	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:    e.Name(),
			Size:    info.Size(),
			IsEntry: e.Name() == entryFile,
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	return files, nil
}

func restoreFromSnapshot(snapshotDir, targetDir string) error {
	os.RemoveAll(targetDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := copyFile(filepath.Join(snapshotDir, e.Name()), filepath.Join(targetDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// createVersionSnapshot 创建版本快照记录 + 拷贝磁盘文件
func (svc *ProgramService) createVersionSnapshot(p *model.Program, helpText, createdBy string) error {
	dir := programDir(svc.workspaceDir, p.Project, p.Name)
	if err := svc.snapshotFiles(p.ID, int(p.CurrentVersion), dir); err != nil {
		return err
	}
	pv := &model.ProgramVersion{
		ProgramID:   p.ID,
		Version:     int(p.CurrentVersion),
		EntryFile:   p.EntryFile,
		Interpreter: p.Interpreter,
		HelpText:    helpText,
		CreatedBy:   createdBy,
	}
	return svc.store.CreateProgramVersion(pv)
}

// snapshotFiles 将当前工作目录拷贝到版本快照目录
func (svc *ProgramService) snapshotFiles(programID uint64, version int, srcDir string) error {
	dstDir := filepath.Join(svc.workspaceDir, ".versions", fmt.Sprintf("%d", programID), fmt.Sprintf("v%d", version))
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("创建快照目录失败: %w", err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("读取源目录失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("拷贝文件 %s 失败: %w", entry.Name(), err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
