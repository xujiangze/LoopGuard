package api

import (
	"LoopGuard/internal/config"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"
	"LoopGuard/web"
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Deps struct {
	Store      *store.Store
	TicketSvc  *service.TicketService
	ProgramSvc *service.ProgramService
	Cfg        config.Config
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	ai := NewAIHandler(d.TicketSvc, d.Cfg)
	human := NewHumanHandler(d.Store, d.TicketSvc, d.Cfg)
	admin := NewAdminHandler(d.Store, d.ProgramSvc, d.Cfg.WorkspaceDir)

	v1 := r.Group("/api/v1")

	v1.POST("/auth/login", human.Login)

	aiGrp := v1.Group("", APIKeyAuth(d.Store))
	aiGrp.POST("/tickets", ai.Submit)

	v1.GET("/programs", APIKeyOrAdminJWTAuth(d.Store, d.Cfg.JWTSecret), admin.ListPrograms)

	// GET /tickets/:id: AI 轮询和人工查看共用，接受 API Key 或 JWT
	v1.GET("/tickets/:id", APIKeyOrJWTAuth(d.Store, d.Cfg.JWTSecret), ai.Get)
	v1.GET("/tickets/:id/executions", JWTAuth(d.Cfg.JWTSecret), human.ListExecutions)

	jwtGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret))
	jwtGrp.POST("/tickets/submit", human.Submit)
	jwtGrp.GET("/tickets", human.ListMine)
	jwtGrp.POST("/tickets/:id/approve", human.Approve)
	jwtGrp.POST("/tickets/:id/reject", human.Reject)

	// Webhook 管理（JWT 认证，admin 或 user 角色）
	webhookHandler := NewWebhookHandler(d.Store)
	jwtGrp.POST("/webhooks", webhookHandler.CreateWebhook)
	jwtGrp.GET("/webhooks", webhookHandler.ListWebhooks)
	jwtGrp.DELETE("/webhooks/:id", webhookHandler.DeleteWebhook)
	jwtGrp.PATCH("/webhooks/:id", webhookHandler.ToggleWebhook)
	jwtGrp.GET("/webhooks/:id/deliveries", webhookHandler.ListDeliveries)

	adminGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret), AdminOnly())
	adminGrp.POST("/programs", admin.CreateProgram)
	adminGrp.PUT("/programs/:id", admin.UpdateProgram)
	adminGrp.DELETE("/programs/:id", admin.DeleteProgram)
	adminGrp.GET("/programs/:id/files", admin.ListFiles)
	adminGrp.GET("/programs/:id/files/:filename", admin.GetFileContent)
	adminGrp.GET("/programs/:id/versions", admin.ListVersions)
	adminGrp.GET("/programs/:id/versions/:version/files", admin.ListVersionFiles)
	adminGrp.GET("/programs/:id/versions/:version/files/:filename", admin.GetVersionFileContent)
	adminGrp.POST("/programs/:id/rollback", admin.Rollback)
	adminGrp.POST("/users", admin.CreateUser)
	adminGrp.GET("/users", admin.ListUsers)
	adminGrp.POST("/api-keys", admin.CreateAPIKey)
	adminGrp.GET("/api-keys", admin.ListAPIKeys)
	adminGrp.PUT("/api-keys/:id", admin.UpdateAPIKey)
	adminGrp.DELETE("/api-keys/:id", admin.DeleteAPIKey)
	adminGrp.PUT("/users/:id/password", admin.ResetPassword)

	// 前端静态文件服务
	distFS, _ := fs.Sub(web.DistFS, "dist")
	r.NoRoute(func(c *gin.Context) {
		// 尝试从嵌入文件系统读取
		path := c.Request.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		f, err := distFS.(fs.ReadFileFS).ReadFile(path[1:])
		if err != nil {
			// SPA fallback: 非 API、非静态资源请求返回 index.html
			idx, _ := distFS.(fs.ReadFileFS).ReadFile("index.html")
			c.Data(http.StatusOK, "text/html; charset=utf-8", idx)
			return
		}
		c.Data(http.StatusOK, mimeByExt(path), f)
	})

	return r
}

func mimeByExt(path string) string {
	switch {
	case len(path) > 3 && path[len(path)-3:] == ".js":
		return "application/javascript"
	case len(path) > 4 && path[len(path)-4:] == ".css":
		return "text/css"
	case len(path) > 5 && path[len(path)-5:] == ".html":
		return "text/html; charset=utf-8"
	case len(path) > 4 && path[len(path)-4:] == ".svg":
		return "image/svg+xml"
	case len(path) > 4 && path[len(path)-4:] == ".png":
		return "image/png"
	case len(path) > 4 && path[len(path)-4:] == ".ico":
		return "image/x-icon"
	case len(path) > 4 && path[len(path)-4:] == ".wasm":
		return "application/wasm"
	case len(path) > 5 && path[len(path)-5:] == ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
