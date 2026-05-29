package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	store    *store.Store
	programs *service.ProgramService
}

func NewAdminHandler(s *store.Store, p *service.ProgramService) *AdminHandler {
	return &AdminHandler{store: s, programs: p}
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := model.RoleUser
	if req.Role == "admin" {
		role = model.RoleAdmin
	}
	hash, _ := auth.HashPassword(req.Password)
	u := &model.User{Username: req.Username, PasswordHash: hash, Role: role}
	if err := h.store.CreateUser(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "username": u.Username, "role": u.Role})
}

func (h *AdminHandler) CreateAPIKey(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plain := auth.GenerateAPIKey()
	k := &model.APIKey{Name: req.Name, KeyHash: auth.HashAPIKey(plain), Enabled: true}
	if err := h.store.CreateAPIKey(k); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": k.ID, "name": k.Name, "api_key": plain})
}

func (h *AdminHandler) CreateProgram(c *gin.Context) {
	var req struct {
		Project      string          `json:"project" binding:"required"`
		Name         string          `json:"name" binding:"required"`
		BinaryPath   string          `json:"binary_path" binding:"required"`
		ApproverID   uint64          `json:"approver_id" binding:"required"`
		TimeoutSec   int             `json:"timeout_sec"`
		ParamsSchema json.RawMessage `json:"params_schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.programs.Register(c.Request.Context(), service.RegisterInput{
		Project: req.Project, Name: req.Name, BinaryPath: req.BinaryPath,
		ApproverID: req.ApproverID, TimeoutSec: req.TimeoutSec, ParamsSchema: req.ParamsSchema,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *AdminHandler) ListPrograms(c *gin.Context) {
	ps, err := h.programs.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ps)
}

func (h *AdminHandler) UpdateProgram(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	p, err := h.store.GetProgram(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	var req struct {
		Enabled    *bool   `json:"enabled"`
		ApproverID *uint64 `json:"approver_id"`
		TimeoutSec *int    `json:"timeout_sec"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	if req.ApproverID != nil {
		p.ApproverID = *req.ApproverID
	}
	if req.TimeoutSec != nil {
		p.TimeoutSec = *req.TimeoutSec
	}
	if err := h.store.UpdateProgram(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}
