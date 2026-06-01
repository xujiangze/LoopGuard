package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/config"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
)

type HumanHandler struct {
	store   *store.Store
	tickets *service.TicketService
	cfg     config.Config
}

func NewHumanHandler(s *store.Store, t *service.TicketService, cfg config.Config) *HumanHandler {
	return &HumanHandler{store: s, tickets: t, cfg: cfg}
}

func (h *HumanHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.store.GetUserByUsername(req.Username)
	if err != nil || !auth.VerifyPassword(u.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	tok, err := auth.SignJWT(h.cfg.JWTSecret, u.ID, string(u.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签发 token 失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "role": u.Role, "user_id": u.ID, "username": u.Username})
}

func (h *HumanHandler) effectiveApprover(c *gin.Context, tk *model.Ticket) uint64 {
	if c.GetString("role") == "admin" {
		return tk.ApproverID
	}
	return c.GetUint64("user_id")
}

func (h *HumanHandler) Approve(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	tk, err := h.store.GetTicket(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	out, err := h.tickets.Approve(c.Request.Context(), id, h.effectiveApprover(c, tk))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *HumanHandler) Reject(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	tk, err := h.store.GetTicket(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	out, err := h.tickets.Reject(id, h.effectiveApprover(c, tk), req.Reason)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *HumanHandler) GetTicket(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	tk, err := h.store.GetTicket(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	c.JSON(http.StatusOK, tk)
}

func (h *HumanHandler) ListExecutions(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	es, err := h.store.ListExecutionsByTicket(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, es)
}

func (h *HumanHandler) Submit(c *gin.Context) {
	var req struct {
		APIKeyID uint64   `json:"api_key_id" binding:"required"`
		Project  string   `json:"project" binding:"required"`
		Name     string   `json:"name" binding:"required"`
		Args     []string `json:"args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	k, err := h.store.GetAPIKey(req.APIKeyID)
	if err != nil || !k.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API Key 不存在或已禁用"})
		return
	}

	tk, err := h.tickets.Submit(c.Request.Context(), service.SubmitInput{
		Project: req.Project, Name: req.Name,
		APIKeyID: req.APIKeyID, Args: req.Args,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{
		"ticket_id":     tk.ID,
		"status":        tk.Status,
		"dryrun_output": tk.DryrunOutput,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HumanHandler) ListMine(c *gin.Context) {
	uid := c.GetUint64("user_id")
	status := model.TicketStatus(c.Query("status"))
	ts, err := h.store.ListTicketsByApprover(uid, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	progIDs := make([]uint64, 0, len(ts))
	keyIDs := make([]uint64, 0, len(ts))
	for _, tk := range ts {
		progIDs = append(progIDs, tk.ProgramID)
		keyIDs = append(keyIDs, tk.SubmittedBy)
	}

	progMap, _ := h.store.GetProgramsByIDs(progIDs)
	keyMap, _ := h.store.GetAPIKeysByIDs(keyIDs)

	items := make([]ticketListItem, len(ts))
	for i, tk := range ts {
		args := make([]string, 0)
		if len(tk.Args) > 0 {
			_ = json.Unmarshal(tk.Args, &args)
		}
		item := ticketListItem{
			ID:          tk.ID,
			ProgramID:   tk.ProgramID,
			Args:        args,
			Status:      tk.Status,
			SubmittedBy: tk.SubmittedBy,
			ApproverID:  tk.ApproverID,
			ApprovedBy:  tk.ApprovedBy,
			ApprovedAt:  tk.ApprovedAt,
			RejectReason: tk.RejectReason,
			CreatedAt:   tk.CreatedAt,
			UpdatedAt:   tk.UpdatedAt,
		}
		if p, ok := progMap[tk.ProgramID]; ok {
			item.ProgramProject = p.Project
			item.ProgramName = p.Name
		}
		if k, ok := keyMap[tk.SubmittedBy]; ok {
			item.SubmittedByName = k.Name
		} else {
			item.SubmittedByName = fmt.Sprintf("Key #%d", tk.SubmittedBy)
		}
		items[i] = item
	}
	c.JSON(http.StatusOK, items)
}

type ticketListItem struct {
	ID              uint64            `json:"id"`
	ProgramID       uint64            `json:"program_id"`
	ProgramProject  string            `json:"program_project"`
	ProgramName     string            `json:"program_name"`
	Args            []string          `json:"args"`
	Status          model.TicketStatus `json:"status"`
	SubmittedBy     uint64            `json:"submitted_by"`
	SubmittedByName string            `json:"submitted_by_name"`
	ApproverID      uint64            `json:"approver_id"`
	ApprovedBy      *uint64           `json:"approved_by"`
	ApprovedAt      *time.Time        `json:"approved_at,omitempty"`
	RejectReason    string            `json:"reject_reason"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}
