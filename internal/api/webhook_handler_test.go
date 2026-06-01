package api

import (
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWebhookAPITest(t *testing.T) (*gin.Engine, *store.Store) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.User{}, &model.WebhookConfig{}, &model.WebhookDelivery{}, &model.Program{})

	s := store.New(db)

	user := &model.User{
		Username:     "admin",
		PasswordHash: "hash",
		Role:         model.RoleAdmin,
	}
	db.Create(user)

	program := &model.Program{
		ID:         1,
		Project:    "test-project",
		Name:       "test-program",
		EntryFile:  "test.sh",
		ApproverID: 1,
	}
	db.Create(program)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	webhookHandler := NewWebhookHandler(s)

	router.POST("/webhooks", webhookHandler.CreateWebhook)
	router.GET("/webhooks", webhookHandler.ListWebhooks)
	router.DELETE("/webhooks/:id", webhookHandler.DeleteWebhook)
	router.PATCH("/webhooks/:id", webhookHandler.ToggleWebhook)
	router.GET("/webhooks/:id/deliveries", webhookHandler.ListDeliveries)

	return router, s
}

func TestCreateWebhook(t *testing.T) {
	router, _ := setupWebhookAPITest(t)

	body := map[string]interface{}{
		"program_id":  1,
		"name":        "Test Webhook",
		"url":         "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		"enabled":     true,
		"event_types": "ticket.pending_approval,ticket.done",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/webhooks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["id"] == nil {
		t.Error("Expected response to contain id")
	}
}

func TestCreateWebhookInvalidURL(t *testing.T) {
	router, _ := setupWebhookAPITest(t)

	body := map[string]interface{}{
		"program_id":  1,
		"name":        "Invalid Webhook",
		"url":         "https://example.com/webhook",
		"enabled":     true,
		"event_types": "ticket.done",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/webhooks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid URL, got %d", w.Code)
	}
}

func TestListWebhooks(t *testing.T) {
	router, s := setupWebhookAPITest(t)

	webhook := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Test Webhook",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.done",
	}
	s.CreateWebhook(webhook)

	req, _ := http.NewRequest("GET", "/webhooks?program_id=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if len(response) != 1 {
		t.Errorf("Expected 1 webhook, got %d", len(response))
	}
}

func TestToggleWebhook(t *testing.T) {
	router, s := setupWebhookAPITest(t)

	webhook := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Toggle Test",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.done",
	}
	s.CreateWebhook(webhook)

	body := map[string]interface{}{"enabled": false}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PATCH", "/webhooks/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["enabled"] != false {
		t.Error("Expected enabled to be false after toggle")
	}
}

func TestToggleWebhookNotFound(t *testing.T) {
	router, _ := setupWebhookAPITest(t)

	body := map[string]interface{}{"enabled": false}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PATCH", "/webhooks/999", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteWebhook(t *testing.T) {
	router, s := setupWebhookAPITest(t)

	webhook := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Delete Test",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.done",
	}
	s.CreateWebhook(webhook)

	req, _ := http.NewRequest("DELETE", "/webhooks/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestListDeliveries(t *testing.T) {
	router, s := setupWebhookAPITest(t)

	webhook := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Delivery Test",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.done",
	}
	s.CreateWebhook(webhook)

	req, _ := http.NewRequest("GET", "/webhooks/1/deliveries", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// 没有 delivery 记录时应返回空数组
	if response == nil {
		t.Error("Expected non-nil empty array")
	}
}
