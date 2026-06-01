package api

import (
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct {
	store *store.Store
}

func NewWebhookHandler(s *store.Store) *WebhookHandler {
	return &WebhookHandler{store: s}
}

type CreateWebhookRequest struct {
	ProgramID  uint64 `json:"program_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	URL        string `json:"url" binding:"required"`
	Enabled    bool   `json:"enabled"`
	EventTypes string `json:"event_types" binding:"required"`
}

type WebhookResponse struct {
	ID         uint64 `json:"id"`
	ProgramID  uint64 `json:"program_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	Enabled    bool   `json:"enabled"`
	EventTypes string `json:"event_types"`
}

// CreateWebhook 创建 Webhook 配置
func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	var req CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// URL 格式验证
	if !strings.Contains(req.URL, "qyapi.weixin.qq.com") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL must contain qyapi.weixin.qq.com"})
		return
	}

	webhook := &model.WebhookConfig{
		ProgramID:  req.ProgramID,
		Name:       req.Name,
		URL:        req.URL,
		Enabled:    req.Enabled,
		EventTypes: req.EventTypes,
	}

	if err := h.store.CreateWebhook(webhook); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, webhook)
}

// ListWebhooks 查询 Webhook 列表
func (h *WebhookHandler) ListWebhooks(c *gin.Context) {
	programID := c.Query("program_id")

	var webhooks []model.WebhookConfig
	var err error

	if programID != "" {
		// 解析 program_id
		var pid uint64
		if _, err := fmt.Sscanf(programID, "%d", &pid); err == nil {
			webhooks, err = h.store.GetWebhooksByProgram(pid)
		}
	} else {
		// 返回所有 Webhook（需要实现 GetAllWebhooks 方法）
		err = h.store.DB().Find(&webhooks).Error
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, webhooks)
}

// DeleteWebhook 删除 Webhook 配置
func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	id := c.Param("id")
	var webhookID uint64
	if _, err := fmt.Sscanf(id, "%d", &webhookID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	if err := h.store.DeleteWebhook(webhookID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted"})
}

// ToggleWebhook 启用/禁用 Webhook
func (h *WebhookHandler) ToggleWebhook(c *gin.Context) {
	id := c.Param("id")
	var webhookID uint64
	if _, err := fmt.Sscanf(id, "%d", &webhookID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	webhook, err := h.store.GetWebhook(webhookID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	webhook.Enabled = req.Enabled
	if err := h.store.UpdateWebhook(webhook); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, webhook)
}

// ListDeliveries 查询 Webhook 投递记录
func (h *WebhookHandler) ListDeliveries(c *gin.Context) {
	id := c.Param("id")
	var webhookID uint64
	if _, err := fmt.Sscanf(id, "%d", &webhookID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	deliveries, err := h.store.GetWebhookDeliveries(webhookID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if deliveries == nil {
		deliveries = []model.WebhookDelivery{}
	}

	c.JSON(http.StatusOK, deliveries)
}
