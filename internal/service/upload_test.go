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
