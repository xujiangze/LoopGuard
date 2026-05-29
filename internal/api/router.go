package api

import (
	"LoopGuard/internal/config"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"
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
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	ai := NewAIHandler(d.TicketSvc, d.Cfg)
	human := NewHumanHandler(d.Store, d.TicketSvc, d.Cfg)
	admin := NewAdminHandler(d.Store, d.ProgramSvc)

	v1 := r.Group("/api/v1")

	v1.POST("/auth/login", human.Login)

	aiGrp := v1.Group("", APIKeyAuth(d.Store))
	aiGrp.POST("/tickets", ai.Submit)

	// GET /tickets/:id: AI 轮询和人工查看共用，接受 API Key 或 JWT
	v1.GET("/tickets/:id", APIKeyOrJWTAuth(d.Store, d.Cfg.JWTSecret), ai.Get)
	v1.GET("/tickets/:id/executions", JWTAuth(d.Cfg.JWTSecret), human.ListExecutions)

	jwtGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret))
	jwtGrp.GET("/tickets", human.ListMine)
	jwtGrp.POST("/tickets/:id/approve", human.Approve)
	jwtGrp.POST("/tickets/:id/reject", human.Reject)

	adminGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret), AdminOnly())
	adminGrp.POST("/programs", admin.CreateProgram)
	adminGrp.GET("/programs", admin.ListPrograms)
	adminGrp.PUT("/programs/:id", admin.UpdateProgram)
	adminGrp.POST("/users", admin.CreateUser)
	adminGrp.GET("/users", admin.ListUsers)
	adminGrp.POST("/api-keys", admin.CreateAPIKey)
	adminGrp.GET("/api-keys", admin.ListAPIKeys)

	return r
}
