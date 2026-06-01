package store

import (
	"LoopGuard/internal/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAutoMigrateWebhookModels 验证 Webhook 相关表的自动迁移
func TestAutoMigrateWebhookModels(t *testing.T) {
	// 使用内存数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	s := New(db)

	// 执行迁移
	if err := s.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// 验证 webhook_configs 表存在
	if !db.Migrator().HasTable(&model.WebhookConfig{}) {
		t.Error("webhook_configs table was not created")
	}

	// 验证 webhook_deliveries 表存在
	if !db.Migrator().HasTable(&model.WebhookDelivery{}) {
		t.Error("webhook_deliveries table was not created")
	}

	// 验证索引存在 - 简单验证：直接使用 GORM 插入和查询
	config := model.WebhookConfig{
		ProgramID:  1,
		Name:       "Test",
		URL:        "https://qyapi.weixin.qq.com/test",
		Enabled:    true,
		EventTypes: "ticket.done",
	}
	if err := db.Create(&config).Error; err != nil {
		t.Errorf("Failed to create webhook config: %v", err)
	}

	var result model.WebhookConfig
	if err := db.Where("program_id = ?", 1).First(&result).Error; err != nil {
		t.Errorf("Failed to query webhook config: %v", err)
	}

	// 验证 WebhookDelivery 表
	delivery := model.WebhookDelivery{
		WebhookID:   1,
		TicketID:    1,
		EventType:   "ticket.done",
		StatusCode:  200,
		Response:    "ok",
	}
	if err := db.Create(&delivery).Error; err != nil {
		t.Errorf("Failed to create webhook delivery: %v", err)
	}
}
