package model

import "time"

// WebhookConfig 企业微信 Webhook 配置
type WebhookConfig struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	ProgramID  uint64    `gorm:"not null;index:idx_program_id" json:"program_id"`
	Name       string    `gorm:"size:128;not null" json:"name"`
	URL        string    `gorm:"size:512;not null" json:"url"`
	Enabled    bool      `gorm:"not null;default:true" json:"enabled"`
	EventTypes string    `gorm:"size:256;not null" json:"event_types"` // 逗号分隔的事件类型
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null" json:"updated_at"`
}

// WebhookDelivery Webhook 投递记录
type WebhookDelivery struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	WebhookID   uint64     `gorm:"not null;index:idx_webhook_id" json:"webhook_id"`
	TicketID    uint64     `gorm:"not null;index:idx_ticket_id" json:"ticket_id"`
	EventType   string     `gorm:"size:64;not null" json:"event_type"`
	StatusCode  int        `gorm:"not null" json:"status_code"`
	Response    string     `gorm:"type:text" json:"response"`
	DeliveredAt *time.Time `json:"delivered_at"`
}
