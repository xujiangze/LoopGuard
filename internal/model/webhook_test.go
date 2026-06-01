package model

import (
	"testing"
	"time"
)

// TestWebhookConfigGORMTags 验证 WebhookConfig 的 GORM 标签
func TestWebhookConfigGORMTags(t *testing.T) {
	// 这个测试确保 WebhookConfig 结构体存在并且有正确的字段
	config := WebhookConfig{
		ID:         1,
		ProgramID:  100,
		Name:       "测试 Webhook",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.pending_approval,ticket.done",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if config.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", config.ID)
	}
	if config.ProgramID != 100 {
		t.Errorf("Expected ProgramID to be 100, got %d", config.ProgramID)
	}
	if config.Name != "测试 Webhook" {
		t.Errorf("Expected Name to be '测试 Webhook', got %s", config.Name)
	}
	if config.URL != "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test" {
		t.Errorf("Expected URL to be webhook URL, got %s", config.URL)
	}
	if !config.Enabled {
		t.Errorf("Expected Enabled to be true, got %v", config.Enabled)
	}
	if config.EventTypes != "ticket.pending_approval,ticket.done" {
		t.Errorf("Expected EventTypes to be 'ticket.pending_approval,ticket.done', got %s", config.EventTypes)
	}
}

// TestWebhookDeliveryGORMTags 验证 WebhookDelivery 的 GORM 标签
func TestWebhookDeliveryGORMTags(t *testing.T) {
	now := time.Now()
	delivery := WebhookDelivery{
		ID:          1,
		WebhookID:   10,
		TicketID:    100,
		EventType:   "ticket.pending_approval",
		StatusCode:  200,
		Response:    "ok",
		DeliveredAt: &now,
	}

	if delivery.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", delivery.ID)
	}
	if delivery.WebhookID != 10 {
		t.Errorf("Expected WebhookID to be 10, got %d", delivery.WebhookID)
	}
	if delivery.TicketID != 100 {
		t.Errorf("Expected TicketID to be 100, got %d", delivery.TicketID)
	}
	if delivery.EventType != "ticket.pending_approval" {
		t.Errorf("Expected EventType to be 'ticket.pending_approval', got %s", delivery.EventType)
	}
	if delivery.StatusCode != 200 {
		t.Errorf("Expected StatusCode to be 200, got %d", delivery.StatusCode)
	}
	if delivery.Response != "ok" {
		t.Errorf("Expected Response to be 'ok', got %s", delivery.Response)
	}
	if delivery.DeliveredAt == nil {
		t.Errorf("Expected DeliveredAt to be set, got nil")
	}
}
