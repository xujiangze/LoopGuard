package store

import (
	"LoopGuard/internal/model"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWebhookTestDB(t *testing.T) *Store {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	s := New(db)
	if err := s.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	return s
}

// TestCreateWebhook 测试创建 Webhook 配置
func TestCreateWebhook(t *testing.T) {
	s := setupWebhookTestDB(t)

	config := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "测试 Webhook",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.pending_approval,ticket.done",
	}

	err := s.CreateWebhook(config)
	if err != nil {
		t.Errorf("CreateWebhook failed: %v", err)
	}

	if config.ID == 0 {
		t.Error("Expected ID to be set after creation")
	}
}

// TestGetWebhooksByProgram 测试按程序查询 Webhook
func TestGetWebhooksByProgram(t *testing.T) {
	s := setupWebhookTestDB(t)

	// 创建测试数据
	config1 := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Webhook 1",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=1",
		Enabled:    true,
		EventTypes: "ticket.pending_approval",
	}
	config2 := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Webhook 2",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=2",
		Enabled:    true,
		EventTypes: "ticket.done",
	}
	config3 := &model.WebhookConfig{
		ProgramID:  2,
		Name:       "Webhook 3",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=3",
		Enabled:    true,
		EventTypes: "ticket.done",
	}

	s.CreateWebhook(config1)
	s.CreateWebhook(config2)
	s.CreateWebhook(config3)

	// 查询程序 1 的 Webhook
	webhooks, err := s.GetWebhooksByProgram(1)
	if err != nil {
		t.Errorf("GetWebhooksByProgram failed: %v", err)
	}

	if len(webhooks) != 2 {
		t.Errorf("Expected 2 webhooks for program 1, got %d", len(webhooks))
	}
}

// TestGetWebhooksByEventType 测试按事件类型查询 Webhook
func TestGetWebhooksByEventType(t *testing.T) {
	s := setupWebhookTestDB(t)

	// 创建测试数据
	config1 := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Webhook 1",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=1",
		Enabled:    true,
		EventTypes: "ticket.pending_approval",
	}
	config2 := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Webhook 2",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=2",
		Enabled:    true,
		EventTypes: "ticket.pending_approval,ticket.done",
	}
	config3 := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Webhook 3",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=3",
		Enabled:    true,
		EventTypes: "ticket.rejected",
	}

	s.CreateWebhook(config1)
	s.CreateWebhook(config2)
	s.CreateWebhook(config3)

	// 查询订阅了 ticket.done 事件的 Webhook
	webhooks, err := s.GetWebhooksByEventType("ticket.done")
	if err != nil {
		t.Errorf("GetWebhooksByEventType failed: %v", err)
	}

	// config2 订阅了 ticket.done，config1 和 config3 没有
	if len(webhooks) != 1 {
		t.Errorf("Expected 1 webhook for ticket.done event, got %d", len(webhooks))
	}

	if len(webhooks) > 0 && webhooks[0].Name != "Webhook 2" {
		t.Errorf("Expected 'Webhook 2', got %s", webhooks[0].Name)
	}
}

// TestDeleteWebhook 测试删除 Webhook
func TestDeleteWebhook(t *testing.T) {
	s := setupWebhookTestDB(t)

	config := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Webhook to delete",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.done",
	}

	s.CreateWebhook(config)

	err := s.DeleteWebhook(config.ID)
	if err != nil {
		t.Errorf("DeleteWebhook failed: %v", err)
	}

	// 验证删除成功
	webhooks, _ := s.GetWebhooksByProgram(1)
	if len(webhooks) != 0 {
		t.Errorf("Expected 0 webhooks after deletion, got %d", len(webhooks))
	}
}

// TestUpdateWebhook 测试更新 Webhook
func TestUpdateWebhook(t *testing.T) {
	s := setupWebhookTestDB(t)

	config := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Original Name",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.done",
	}

	s.CreateWebhook(config)

	// 更新
	config.Name = "Updated Name"
	config.Enabled = false

	err := s.UpdateWebhook(config)
	if err != nil {
		t.Errorf("UpdateWebhook failed: %v", err)
	}

	// 验证更新成功
	webhooks, _ := s.GetWebhooksByProgram(1)
	if len(webhooks) != 1 {
		t.Errorf("Expected 1 webhook after update, got %d", len(webhooks))
	}

	if webhooks[0].Name != "Updated Name" {
		t.Errorf("Expected name to be 'Updated Name', got %s", webhooks[0].Name)
	}

	if webhooks[0].Enabled != false {
		t.Errorf("Expected Enabled to be false, got %v", webhooks[0].Enabled)
	}
}

// TestCreateWebhookDelivery 测试创建投递记录
func TestCreateWebhookDelivery(t *testing.T) {
	s := setupWebhookTestDB(t)

	now := time.Now()
	delivery := &model.WebhookDelivery{
		WebhookID:   1,
		TicketID:    100,
		EventType:   "ticket.pending_approval",
		StatusCode:  200,
		Response:    "ok",
		DeliveredAt: &now,
	}

	err := s.CreateWebhookDelivery(delivery)
	if err != nil {
		t.Errorf("CreateWebhookDelivery failed: %v", err)
	}

	if delivery.ID == 0 {
		t.Error("Expected ID to be set after creation")
	}
}

// TestGetWebhookDeliveries 测试查询投递记录
func TestGetWebhookDeliveries(t *testing.T) {
	s := setupWebhookTestDB(t)

	// 创建测试数据
	now := time.Now()
	delivery1 := &model.WebhookDelivery{
		WebhookID:   1,
		TicketID:    100,
		EventType:   "ticket.pending_approval",
		StatusCode:  200,
		Response:    "ok",
		DeliveredAt: &now,
	}
	delivery2 := &model.WebhookDelivery{
		WebhookID:   1,
		TicketID:    101,
		EventType:   "ticket.done",
		StatusCode:  200,
		Response:    "ok",
		DeliveredAt: &now,
	}

	s.CreateWebhookDelivery(delivery1)
	s.CreateWebhookDelivery(delivery2)

	// 查询 webhook 1 的投递记录
	deliveries, err := s.GetWebhookDeliveries(1)
	if err != nil {
		t.Errorf("GetWebhookDeliveries failed: %v", err)
	}

	if len(deliveries) != 2 {
		t.Errorf("Expected 2 deliveries, got %d", len(deliveries))
	}

	// 验证按 ID 降序排列
	if deliveries[0].ID < deliveries[1].ID {
		t.Error("Expected deliveries to be ordered by ID desc")
	}
}
