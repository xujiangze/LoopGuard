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

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []model.User
	if err := h.store.DB().Order("id asc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) ListAPIKeys(c *gin.Context) {
	var keys []model.APIKey
	if err := h.store.DB().Order("id desc").Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, keys)
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
		Interpreter  string          `json:"interpreter"`
		ApproverID   uint64          `json:"approver_id" binding:"required"`
		TimeoutSec   int             `json:"timeout_sec"`
		ParamsSchema json.RawMessage `json:"params_schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.programs.Register(c.Request.Context(), service.RegisterInput{
		Project: req.Project, Name: req.Name, BinaryPath: req.BinaryPath, Interpreter: req.Interpreter,
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
		Enabled     *bool   `json:"enabled"`
		Interpreter *string `json:"interpreter"`
		ApproverID  *uint64 `json:"approver_id"`
		TimeoutSec  *int    `json:"timeout_sec"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	if req.Interpreter != nil {
		p.Interpreter = *req.Interpreter
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

func (h *AdminHandler) UpdateAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var k model.APIKey
	if err := h.store.DB().First(&k, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API Key 不存在"})
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Enabled != nil {
		k.Enabled = *req.Enabled
	}
	if err := h.store.UpdateAPIKey(&k); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, k)
}

func (h *AdminHandler) DeleteAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.store.DeleteAPIKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (h *AdminHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, _ := auth.HashPassword(req.Password)
	if err := h.store.UpdateUserPassword(id, hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码已重置"})
}
