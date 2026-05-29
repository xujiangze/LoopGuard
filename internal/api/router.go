package api

import (
	"LoopGuard/internal/config"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

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

	ai := NewAIHandler(d.TicketSvc, d.Cfg)
	human := NewHumanHandler(d.Store, d.TicketSvc, d.Cfg)
	admin := NewAdminHandler(d.Store, d.ProgramSvc)

	v1 := r.Group("/api/v1")

	v1.POST("/auth/login", human.Login)

	aiGrp := v1.Group("", APIKeyAuth(d.Store))
	aiGrp.POST("/tickets", ai.Submit)
	aiGrp.GET("/tickets/:id", ai.Get)

	jwtGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret))
	jwtGrp.GET("/tickets", human.ListMine)
	jwtGrp.POST("/tickets/:id/approve", human.Approve)
	jwtGrp.POST("/tickets/:id/reject", human.Reject)

	adminGrp := v1.Group("", JWTAuth(d.Cfg.JWTSecret), AdminOnly())
	adminGrp.POST("/programs", admin.CreateProgram)
	adminGrp.GET("/programs", admin.ListPrograms)
	adminGrp.PUT("/programs/:id", admin.UpdateProgram)
	adminGrp.POST("/users", admin.CreateUser)
	adminGrp.POST("/api-keys", admin.CreateAPIKey)

	return r
}
