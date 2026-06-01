package service

import (
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gorm.io/datatypes"
)

// WebhookService 企业微信 Webhook 服务
type WebhookService struct {
	store *store.Store
}

// NewWebhookService 创建 WebhookService 实例
func NewWebhookService(s *store.Store) *WebhookService {
	return &WebhookService{store: s}
}

// WechatMessage 企业微信消息格式
type WechatMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

// Trigger 触发 Webhook 发送（异步）
func (s *WebhookService) Trigger(ticketID uint64, eventType model.TicketStatus, approvalURL string) {
	go func() {
		s.send(ticketID, eventType, approvalURL)
	}()
}

// send 实际发送 Webhook
func (s *WebhookService) send(ticketID uint64, eventType model.TicketStatus, approvalURL string) {
	// 查询工单信息
	db := s.store.DB()
	var ticket model.Ticket
	if err := db.Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		return
	}

	// 查询程序信息
	var program model.Program
	if err := db.Where("id = ?", ticket.ProgramID).First(&program).Error; err != nil {
		return
	}

	// 查询订阅了该事件的 Webhook
	webhooks, err := s.store.GetWebhooksByEventType(string(eventType))
	if err != nil || len(webhooks) == 0 {
		return
	}

	// 遍历发送
	for _, webhook := range webhooks {
		if !webhook.Enabled {
			continue
		}

		// 检查事件类型是否匹配
		if !strings.Contains(webhook.EventTypes, string(eventType)) {
			continue
		}

		// 构建消息
		message := s.buildMessage(&ticket, &program, approvalURL)

		// 发送 HTTP 请求
		statusCode, response := s.sendHTTP(webhook.URL, message)

		// 记录投递
		now := time.Now()
		delivery := &model.WebhookDelivery{
			WebhookID:   webhook.ID,
			TicketID:    ticketID,
			EventType:  string(eventType),
			StatusCode: statusCode,
			Response:   response,
			DeliveredAt: &now,
		}
		s.store.CreateWebhookDelivery(delivery)
	}
}

// sendHTTP 发送 HTTP POST 请求
func (s *WebhookService) sendHTTP(url string, message WechatMessage) (int, string) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return 0, fmt.Sprintf("JSON marshal error: %v", err)
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Sprintf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	responseBody := ""
	if resp.Body != nil {
		buf := new(strings.Builder)
		io.Copy(buf, resp.Body)
		responseBody = buf.String()
	}

	return resp.StatusCode, responseBody
}

// buildMessage 构建企业微信消息
func (s *WebhookService) buildMessage(ticket *model.Ticket, program *model.Program, approvalURL string) WechatMessage {
	content := fmt.Sprintf("LoopGuard 工单通知\n\n程序: %s/%s\n工单ID: %d\n状态: %s\n参数: %s\n\n审批链接: %s",
		program.Project,
		program.Name,
		ticket.ID,
		ticket.Status,
		s.formatTicketContent(ticket.Args),
		approvalURL,
	)

	var message WechatMessage
	message.MsgType = "text"
	message.Text.Content = content
	return message
}

// formatTicketContent 格式化工单参数
func (s *WebhookService) formatTicketContent(args datatypes.JSON) string {
	var params map[string]interface{}
	if err := json.Unmarshal(args, &params); err != nil {
		return string(args)
	}

	var lines []string
	for k, v := range params {
		lines = append(lines, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(lines, "\n")
}
