package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// setupE2ETest 初始化完整的 E2E 测试环境：DB、Store、路由、WebhookService
func setupE2ETest(t *testing.T) (*gin.Engine, *store.Store, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db.AutoMigrate(
		&model.User{}, &model.WebhookConfig{}, &model.WebhookDelivery{},
		&model.Program{}, &model.Ticket{}, &model.Execution{},
	)

	// 创建审批人
	user := &model.User{
		Username:     "approver",
		PasswordHash: "hash",
		Role:         model.RoleAdmin,
	}
	db.Create(user)

	// 创建程序
	program := &model.Program{
		Project:    "e2e-project",
		Name:       "e2e-program",
		EntryFile:  "test.sh",
		ApproverID: 1,
	}
	db.Create(program)

	s := store.New(db)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	webhookHandler := NewWebhookHandler(s)

	router.POST("/webhooks", webhookHandler.CreateWebhook)
	router.GET("/webhooks", webhookHandler.ListWebhooks)
	router.DELETE("/webhooks/:id", webhookHandler.DeleteWebhook)
	router.PATCH("/webhooks/:id", webhookHandler.ToggleWebhook)
	router.GET("/webhooks/:id/deliveries", webhookHandler.ListDeliveries)

	return router, s, db
}

// doRequest 辅助函数：发送 HTTP 请求并返回响应
func doRequest(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, path, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ---------- 测试 1: Webhook 创建 E2E ----------

func TestE2E_WebhookCreation(t *testing.T) {
	router, _, _ := setupE2ETest(t)

	t.Run("successful_create", func(t *testing.T) {
		body := map[string]interface{}{
			"program_id":  1,
			"name":        "E2E Webhook",
			"url":         "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=e2e",
			"enabled":     true,
			"event_types": "ticket.pending_approval,ticket.done",
		}

		w := doRequest(t, router, "POST", "/webhooks", body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d; body: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if resp["id"] == nil {
			t.Error("Expected response to contain id")
		}
		if resp["name"] != "E2E Webhook" {
			t.Errorf("Expected name='E2E Webhook', got %v", resp["name"])
		}
	})

	t.Run("invalid_url_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"program_id":  1,
			"name":        "Bad URL Webhook",
			"url":         "https://example.com/webhook",
			"enabled":     true,
			"event_types": "ticket.done",
		}

		w := doRequest(t, router, "POST", "/webhooks", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for non-qyapi URL, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if !strings.Contains(resp["error"].(string), "qyapi.weixin.qq.com") {
			t.Errorf("Expected error to mention qyapi.weixin.qq.com, got: %v", resp["error"])
		}
	})

	t.Run("missing_required_fields", func(t *testing.T) {
		// 缺少 url 和 event_types
		body := map[string]interface{}{
			"program_id": 1,
			"name":       "Missing Fields",
		}

		w := doRequest(t, router, "POST", "/webhooks", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing fields, got %d", w.Code)
		}
	})

	t.Run("create_then_list", func(t *testing.T) {
		// 先创建
		body := map[string]interface{}{
			"program_id":  1,
			"name":        "List Test Webhook",
			"url":         "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=list-test",
			"enabled":     true,
			"event_types": "ticket.pending_approval",
		}
		w := doRequest(t, router, "POST", "/webhooks", body)
		if w.Code != http.StatusOK {
			t.Fatalf("Create failed: %d", w.Code)
		}

		// 再查询
		w = doRequest(t, router, "GET", "/webhooks?program_id=1", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List failed: %d", w.Code)
		}

		var list []map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
			t.Fatalf("Failed to parse list response: %v", err)
		}

		if len(list) < 1 {
			t.Errorf("Expected at least 1 webhook for program_id=1, got %d", len(list))
		}

		// 验证刚创建的 webhook 在结果中
		found := false
		for _, wh := range list {
			if wh["name"] == "List Test Webhook" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find 'List Test Webhook' in list results")
		}
	})
}

// ---------- 测试 2: Webhook 触发 E2E ----------

func TestE2E_WebhookTrigger(t *testing.T) {
	// 准备 mock webhook 接收服务器
	receivedCh := make(chan map[string]interface{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)
		receivedCh <- payload
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	// 初始化数据库和 store
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db.AutoMigrate(
		&model.User{}, &model.WebhookConfig{}, &model.WebhookDelivery{},
		&model.Program{}, &model.Ticket{}, &model.Execution{},
	)

	user := &model.User{
		Username:     "approver",
		PasswordHash: "hash",
		Role:         model.RoleAdmin,
	}
	db.Create(user)

	program := &model.Program{
		Project:    "trigger-project",
		Name:       "trigger-program",
		EntryFile:  "trigger.sh",
		ApproverID: 1,
	}
	db.Create(program)

	s := store.New(db)

	// 创建 webhook 配置，URL 指向 mock server
	webhook := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Trigger Test Webhook",
		URL:        server.URL,
		Enabled:    true,
		EventTypes: "ticket.pending_approval,ticket.done",
	}
	if err := s.CreateWebhook(webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// 创建工单
	ticket := &model.Ticket{
		ProgramID:  1,
		Status:     model.StatusPendingApproval,
		Args:       datatypes.JSON([]byte(`["--env=staging","--force"]`)),
		ApproverID: 1,
	}
	if err := s.CreateTicket(ticket); err != nil {
		t.Fatalf("Failed to create ticket: %v", err)
	}

	// 通过 WebhookService 触发
	webhookSvc := service.NewWebhookService(s)
	webhookSvc.Trigger(ticket.ID, model.StatusPendingApproval, "http://localhost:8080/tickets/"+fmt.Sprintf("%d", ticket.ID))

	// 等待异步 goroutine 完成
	select {
	case payload := <-receivedCh:
		// 验证消息格式
		if payload["msgtype"] != "text" {
			t.Errorf("Expected msgtype='text', got %v", payload["msgtype"])
		}

		textObj, ok := payload["text"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected 'text' to be an object, got %T", payload["text"])
		}
		content, ok := textObj["content"].(string)
		if !ok {
			t.Fatalf("Expected 'content' to be a string, got %T", textObj["content"])
		}

		// 验证消息内容包含关键信息
		checks := []struct{ label, want string }{
			{"工单ID", fmt.Sprintf("%d", ticket.ID)},
			{"状态", "pending_approval"},
			{"程序名", "trigger-project/trigger-program"},
			{"审批链接", fmt.Sprintf("http://localhost:8080/tickets/%d", ticket.ID)},
		}
		for _, c := range checks {
			if !strings.Contains(content, c.want) {
				t.Errorf("Expected content to contain %s '%s', got: %s", c.label, c.want, content)
			}
		}

	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for webhook delivery")
	}

	// 验证投递记录被正确创建
	time.Sleep(100 * time.Millisecond)

	var deliveries []model.WebhookDelivery
	db.Where("webhook_id = ? AND ticket_id = ?", webhook.ID, ticket.ID).Find(&deliveries)
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery record, got %d", len(deliveries))
	}

	d := deliveries[0]
	if d.StatusCode != 200 {
		t.Errorf("Expected delivery status_code=200, got %d", d.StatusCode)
	}
	if d.EventType != string(model.StatusPendingApproval) {
		t.Errorf("Expected event_type='pending_approval', got %s", d.EventType)
	}
	if !strings.Contains(d.Response, "ok") {
		t.Errorf("Expected response to contain 'ok', got: %s", d.Response)
	}
}

// ---------- 测试 3: 投递记录查询 E2E ----------

func TestE2E_DeliveryQuery(t *testing.T) {
	// 准备 mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	// 初始化
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db.AutoMigrate(
		&model.User{}, &model.WebhookConfig{}, &model.WebhookDelivery{},
		&model.Program{}, &model.Ticket{}, &model.Execution{},
	)

	user := &model.User{
		Username:     "approver",
		PasswordHash: "hash",
		Role:         model.RoleAdmin,
	}
	db.Create(user)

	program := &model.Program{
		Project:    "delivery-project",
		Name:       "delivery-program",
		EntryFile:  "delivery.sh",
		ApproverID: 1,
	}
	db.Create(program)

	s := store.New(db)

	// 创建 webhook
	webhook := &model.WebhookConfig{
		ProgramID:  1,
		Name:       "Delivery Query Webhook",
		URL:        server.URL,
		Enabled:    true,
		EventTypes: "ticket.pending_approval",
	}
	s.CreateWebhook(webhook)

	// 创建工单
	ticket := &model.Ticket{
		ProgramID:  1,
		Status:     model.StatusPendingApproval,
		Args:       datatypes.JSON([]byte(`["--test"]`)),
		ApproverID: 1,
	}
	s.CreateTicket(ticket)

	// 触发 webhook 产生投递记录
	webhookSvc := service.NewWebhookService(s)
	webhookSvc.Trigger(ticket.ID, model.StatusPendingApproval, "http://localhost:8080/tickets/"+fmt.Sprintf("%d", ticket.ID))

	// 等待异步投递完成
	time.Sleep(300 * time.Millisecond)

	// 设置路由用于查询投递记录
	gin.SetMode(gin.TestMode)
	router := gin.New()
	webhookHandler := NewWebhookHandler(s)
	router.GET("/webhooks/:id/deliveries", webhookHandler.ListDeliveries)

	// 查询投递记录
	w := doRequest(t, router, "GET", "/webhooks/1/deliveries", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var deliveries []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &deliveries); err != nil {
		t.Fatalf("Failed to parse deliveries response: %v", err)
	}

	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}

	d := deliveries[0]

	// 验证投递记录字段
	if d["webhook_id"] == nil {
		t.Error("Expected delivery to have webhook_id")
	}
	if d["ticket_id"] == nil {
		t.Error("Expected delivery to have ticket_id")
	}
	if d["event_type"] != "pending_approval" {
		t.Errorf("Expected event_type='pending_approval', got %v", d["event_type"])
	}
	if d["status_code"] == nil {
		t.Error("Expected delivery to have status_code")
	}

	// status_code 在 JSON 反序列化后是 float64
	statusCode, ok := d["status_code"].(float64)
	if !ok {
		t.Errorf("Expected status_code to be numeric, got %T", d["status_code"])
	} else if statusCode != 200 {
		t.Errorf("Expected status_code=200, got %d", int(statusCode))
	}

	if d["response"] == nil {
		t.Error("Expected delivery to have response")
	}
}
