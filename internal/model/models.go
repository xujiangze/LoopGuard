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
	SubmittedBy  uint64         `gorm:"not null" json:"submitted_by"`
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
