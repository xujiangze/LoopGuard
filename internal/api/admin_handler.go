package api

import (
	"net/http"
	"strconv"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	store        *store.Store
	programs     *service.ProgramService
	workspaceDir string
}

func NewAdminHandler(s *store.Store, p *service.ProgramService, workspaceDir string) *AdminHandler {
	return &AdminHandler{store: s, programs: p, workspaceDir: workspaceDir}
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
	project := c.PostForm("project")
	name := c.PostForm("name")
	entryFile := c.PostForm("entry_file")
	interpreter := c.PostForm("interpreter")
	approverIDStr := c.PostForm("approver_id")
	timeoutSecStr := c.PostForm("timeout_sec")

	if project == "" || name == "" || entryFile == "" || interpreter == "" || approverIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project/name/entry_file/interpreter/approver_id 必填"})
		return
	}
	approverID, _ := strconv.ParseUint(approverIDStr, 10, 64)
	timeoutSec, _ := strconv.Atoi(timeoutSecStr)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart 表单解析失败"})
		return
	}
	files := form.File["files"]

	p, err := h.programs.Register(c.Request.Context(), service.RegisterInput{
		Project: project, Name: name, EntryFile: entryFile, Interpreter: interpreter,
		ApproverID: approverID, TimeoutSec: timeoutSec, Files: files,
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

	var in service.UpdateInput

	if v := c.PostForm("entry_file"); v != "" {
		in.EntryFile = &v
	}
	if v := c.PostForm("interpreter"); v != "" {
		in.Interpreter = &v
	}
	if v := c.PostForm("approver_id"); v != "" {
		n, _ := strconv.ParseUint(v, 10, 64)
		in.ApproverID = &n
	}
	if v := c.PostForm("timeout_sec"); v != "" {
		n, _ := strconv.Atoi(v)
		in.TimeoutSec = &n
	}
	if v := c.PostForm("enabled"); v != "" {
		b := v == "true"
		in.Enabled = &b
	}

	form, err := c.MultipartForm()
	if err == nil {
		in.Files = form.File["files"]
	}

	p, err := h.programs.Update(c.Request.Context(), id, in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
