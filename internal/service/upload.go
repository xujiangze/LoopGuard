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
)

const (
	maxFileSize  = 10 << 20 // 10MB
	maxFileCount = 20
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
