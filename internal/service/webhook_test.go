package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func mockWebhookServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handler))
}

func setupWebhookServiceTest(t *testing.T) (*WebhookService, *store.Store, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.AutoMigrate(&model.WebhookConfig{}, &model.WebhookDelivery{}, &model.Ticket{}, &model.Program{})

	program := &model.Program{
		ID:         1,
		Project:    "test-project",
		Name:       "test-program",
		EntryFile:  "test.sh",
		ApproverID: 1,
	}
	db.Create(program)

	webhook := &model.WebhookConfig{
		ID:         1,
		ProgramID:  1,
		Name:       "Test Webhook",
		URL:        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		Enabled:    true,
		EventTypes: "ticket.pending_approval,ticket.done",
	}
	db.Create(webhook)

	s := store.New(db)
	service := NewWebhookService(s)
	return service, s, db
}

func TestNewWebhookService(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s := store.New(db)
	service := NewWebhookService(s)

	if service == nil {
		t.Error("Expected service to be created")
	}
	if service.store == nil {
		t.Error("Expected store to be initialized")
	}
}

func TestTrigger(t *testing.T) {
	service, _, db := setupWebhookServiceTest(t)

	requestReceived := false
	server := mockWebhookServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true

		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("Failed to parse request body: %v", err)
		}
		if payload["msgtype"] != "text" {
			t.Errorf("Expected msgtype to be 'text', got %v", payload["msgtype"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode": 0, "errmsg": "ok"}`))
	})
	defer server.Close()

	db.Model(&model.WebhookConfig{}).Where("id = ?", 1).Update("url", server.URL)

	ticket := &model.Ticket{
		ID:        100,
		ProgramID: 1,
		Status:    model.StatusPendingApproval,
		Args:      datatypes.JSON([]byte(`{"arg1":"value1"}`)),
	}
	db.Create(ticket)

	service.Trigger(ticket.ID, model.StatusPendingApproval, "http://localhost:8080/tickets/100")

	time.Sleep(100 * time.Millisecond)

	if !requestReceived {
		t.Error("Expected webhook request to be sent")
	}

	var deliveries []model.WebhookDelivery
	db.Where("webhook_id = ? AND ticket_id = ?", 1, 100).Find(&deliveries)
	if len(deliveries) != 1 {
		t.Errorf("Expected 1 delivery record, got %d", len(deliveries))
	}
	if len(deliveries) > 0 && deliveries[0].StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", deliveries[0].StatusCode)
	}
}

func TestBuildMessage(t *testing.T) {
	service, _, db := setupWebhookServiceTest(t)

	ticket := &model.Ticket{
		ID:        100,
		ProgramID: 1,
		Status:    model.StatusPendingApproval,
		Args:      datatypes.JSON([]byte(`{"arg1":"value1","arg2":"value2"}`)),
	}

	var program model.Program
	db.First(&program, 1)

	message := service.buildMessage(ticket, &program, "http://localhost:8080/tickets/100")

	if message.MsgType != "text" {
		t.Errorf("Expected msgtype to be 'text', got %s", message.MsgType)
	}

	content := message.Text.Content
	requiredStrings := []string{
		"LoopGuard 工单通知",
		"test-project/test-program",
		"100",
		"pending_approval",
		"arg1=value1",
		"http://localhost:8080/tickets/100",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(content, s) {
			t.Errorf("Expected content to contain '%s', got: %s", s, content)
		}
	}
}

func TestFormatTicketContent(t *testing.T) {
	service, _, _ := setupWebhookServiceTest(t)

	args := datatypes.JSON([]byte(`{"arg1":"value1","arg2":"value2","flag":""}`))
	content := service.formatTicketContent(args)

	if !strings.Contains(content, "arg1=value1") {
		t.Errorf("Expected content to contain 'arg1=value1', got: %s", content)
	}
	if !strings.Contains(content, "arg2=value2") {
		t.Errorf("Expected content to contain 'arg2=value2', got: %s", content)
	}
}

func TestTriggerWebhookNotEnabled(t *testing.T) {
	service, _, db := setupWebhookServiceTest(t)

	requestReceived := false
	server := mockWebhookServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	db.Model(&model.WebhookConfig{}).Where("id = ?", 1).Update("enabled", false)

	ticket := &model.Ticket{
		ID:        100,
		ProgramID: 1,
		Status:    model.StatusPendingApproval,
		Args:      datatypes.JSON([]byte(`{"arg1":"value1"}`)),
	}
	db.Create(ticket)

	service.Trigger(ticket.ID, model.StatusPendingApproval, "http://localhost:8080/tickets/100")

	time.Sleep(100 * time.Millisecond)

	if requestReceived {
		t.Error("Expected webhook request NOT to be sent for disabled webhook")
	}
}

func TestTriggerWebhookNotMatchingEventType(t *testing.T) {
	service, _, db := setupWebhookServiceTest(t)

	requestReceived := false
	server := mockWebhookServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	db.Model(&model.WebhookConfig{}).Where("id = ?", 1).Update("event_types", "ticket.done")

	ticket := &model.Ticket{
		ID:        100,
		ProgramID: 1,
		Status:    model.StatusRejected,
		Args:      datatypes.JSON([]byte(`{"arg1":"value1"}`)),
	}
	db.Create(ticket)

	service.Trigger(ticket.ID, model.StatusRejected, "http://localhost:8080/tickets/100")

	time.Sleep(100 * time.Millisecond)

	if requestReceived {
		t.Error("Expected webhook request NOT to be sent when event type doesn't match")
	}
}

// ---------- 追加的 3 个测试 ----------

// TestWechatMessageFormat 验证企业微信通知的请求格式和投递记录
func TestWechatMessageFormat(t *testing.T) {
	service, _, db := setupWebhookServiceTest(t)

	var receivedContentType string
	var receivedBody map[string]interface{}

	server := mockWebhookServer(t, func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode": 0, "errmsg": "ok"}`))
	})
	defer server.Close()

	// 将 webhook URL 指向 mock server
	db.Model(&model.WebhookConfig{}).Where("id = ?", 1).Update("url", server.URL)

	ticket := &model.Ticket{
		ID:        200,
		ProgramID: 1,
		Status:    model.StatusPendingApproval,
		Args:      datatypes.JSON([]byte(`{"env":"production"}`)),
	}
	db.Create(ticket)

	service.Trigger(ticket.ID, model.StatusPendingApproval, "http://localhost:8080/tickets/200")

	// 等待异步发送完成
	time.Sleep(200 * time.Millisecond)

	// 验证 Content-Type
	if receivedContentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", receivedContentType)
	}

	// 验证 msgtype
	if receivedBody["msgtype"] != "text" {
		t.Errorf("Expected msgtype 'text', got '%v'", receivedBody["msgtype"])
	}

	// 验证 text.content 包含关键信息
	textField, ok := receivedBody["text"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'text' field to be a JSON object")
	}
	content, _ := textField["content"].(string)

	requiredKeywords := []string{
		"LoopGuard 工单通知",
		"test-program", // 程序名
		"200",          // 工单 ID
		"pending_approval",
		"http://localhost:8080/tickets/200", // 审批链接
	}
	for _, kw := range requiredKeywords {
		if !strings.Contains(content, kw) {
			t.Errorf("Expected text.content to contain '%s', got: %s", kw, content)
		}
	}

	// 验证数据库中创建了投递记录，StatusCode = 200
	var deliveries []model.WebhookDelivery
	db.Where("webhook_id = ? AND ticket_id = ?", 1, 200).Find(&deliveries)
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery record, got %d", len(deliveries))
	}
	if deliveries[0].StatusCode != 200 {
		t.Errorf("Expected delivery StatusCode 200, got %d", deliveries[0].StatusCode)
	}
}

// TestTriggerDoesNotBlock 验证 Trigger 是异步发送，不阻塞调用方
func TestTriggerDoesNotBlock(t *testing.T) {
	service, _, db := setupWebhookServiceTest(t)

	requestReceived := make(chan struct{}, 1)

	// 创建一个会延迟 2 秒才响应的 mock server
	server := mockWebhookServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode": 0}`))
		select {
		case requestReceived <- struct{}{}:
		default:
		}
	})
	defer server.Close()

	db.Model(&model.WebhookConfig{}).Where("id = ?", 1).Update("url", server.URL)

	ticket := &model.Ticket{
		ID:        300,
		ProgramID: 1,
		Status:    model.StatusPendingApproval,
		Args:      datatypes.JSON([]byte(`{"arg1":"value1"}`)),
	}
	db.Create(ticket)

	start := time.Now()
	service.Trigger(ticket.ID, model.StatusPendingApproval, "http://localhost:8080/tickets/300")
	elapsed := time.Since(start)

	// Trigger 应在 100ms 内返回（证明是异步的）
	if elapsed > 100*time.Millisecond {
		t.Errorf("Trigger took %v to return, expected < 100ms (async)", elapsed)
	}

	// 等待 mock server 收到请求（最多 5 秒）
	select {
	case <-requestReceived:
		// 成功收到请求
	case <-time.After(5 * time.Second):
		t.Error("Timed out waiting for mock server to receive request")
	}
}

// TestWebhookFailureDoesNotAffectTicket 验证 Webhook 发送失败不影响工单提交流程
func TestWebhookFailureDoesNotAffectTicket(t *testing.T) {
	// 创建会返回 500 的 mock server
	server := mockWebhookServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errcode": 1, "errmsg": "internal error"}`))
	})
	defer server.Close()

	// 使用 ticket_test 的辅助函数创建 TicketService
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command:  "bash deploy.sh -e prod --only-print",
		ExitCode: 0,
		Stdout:   "DRYRUN-OK",
		Stderr:   "",
	}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s)

	// 创建 WebhookService 并注入到 TicketService
	ws := NewWebhookService(s)

	// 手动插入 webhook 配置，URL 指向 mock server
	s.DB().Create(&model.WebhookConfig{
		ProgramID:  p.ID,
		Name:       "Test Webhook",
		URL:        server.URL,
		Enabled:    true,
		EventTypes: "pending_approval",
	})

	svc.SetWebhook(ws, "http://localhost:8080")

	// 提交工单
	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project:  "demo",
		Name:     "deploy",
		APIKeyID: 7,
		Args:     []string{"-e", "prod"},
	})
	if err != nil {
		t.Fatalf("Submit should not fail due to webhook error, got: %v", err)
	}

	// 验证工单状态正确（pending_approval）
	if tk.Status != model.StatusPendingApproval {
		t.Errorf("Expected ticket status 'pending_approval', got '%s'", tk.Status)
	}

	// 等待异步 webhook 发送完成
	time.Sleep(300 * time.Millisecond)

	// 验证 webhook 投递记录中记录了失败状态码
	var deliveries []model.WebhookDelivery
	s.DB().Where("ticket_id = ?", tk.ID).Find(&deliveries)
	if len(deliveries) == 0 {
		t.Fatal("Expected at least 1 delivery record, got 0")
	}
	found := false
	for _, d := range deliveries {
		if d.StatusCode == 500 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected a delivery record with StatusCode 500, got: %+v", deliveries)
	}
}
