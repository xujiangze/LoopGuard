package api

import (
	"fmt"
	"net/http"
	"strconv"

	"LoopGuard/internal/config"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	tickets *service.TicketService
	cfg     config.Config
}

func NewAIHandler(t *service.TicketService, cfg config.Config) *AIHandler {
	return &AIHandler{tickets: t, cfg: cfg}
}

type submitReq struct {
	Project string         `json:"project" binding:"required"`
	Name    string         `json:"name" binding:"required"`
	Args    map[string]any `json:"args"`
}

func (h *AIHandler) Submit(c *gin.Context) {
	var req submitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	apiKeyID := c.GetUint64("api_key_id")
	tk, err := h.tickets.Submit(c.Request.Context(), service.SubmitInput{
		Project: req.Project, Name: req.Name, APIKeyID: apiKeyID, Args: req.Args,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	url := fmt.Sprintf("%s/tickets/%d", h.cfg.BaseURL, tk.ID)
	resp := gin.H{"ticket_id": tk.ID, "status": tk.Status, "approval_url": url}
	switch tk.Status {
	case model.StatusPendingApproval:
		resp["next_action"] = fmt.Sprintf(
			"任务已提交，需人工审批。请通知用户访问审批链接 %s 找审批人审批。审批通过后任务将自动执行，你可轮询 GET /api/v1/tickets/%d 获取结果。",
			url, tk.ID)
		resp["dryrun_output"] = tk.DryrunOutput
	case model.StatusDryrunFailed:
		resp["next_action"] = "dry-run 校验未通过，任务未进入审批。请检查程序是否正确实现 --only-print（需输出 DRYRUN-OK 且退出码为 0）。详情见 dryrun_output。"
		resp["dryrun_output"] = tk.DryrunOutput
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AIHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 id"})
		return
	}
	tk, err := h.tickets.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	c.JSON(http.StatusOK, tk)
}
