package api

import (
	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/handler"
	"github.com/indugate/gateway/internal/api/middleware"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RouterDeps struct {
	DriverManager    *service.DriverManager
	SimulatorService *service.SimulatorService
}

func NewRouter(cfg *config.Config, log *zap.Logger, db *gorm.DB, deps RouterDeps) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logger(log))
	r.Use(middleware.CORS())

	healthHandler := handler.NewHealthHandler()
	deviceService := service.NewDeviceService(db, deps.DriverManager)
	deviceHandler := handler.NewDeviceHandler(deviceService)
	deviceDataHandler := handler.NewDeviceDataHandler(deviceService)
	simulatorHandler := handler.NewSimulatorHandler(deps.SimulatorService)
	mcpHandler := handler.NewMCPHandler(cfg.MCP.BasePath)

	r.GET("/health", healthHandler.Check)

	v1 := r.Group("/api/v1")
	{
		devices := v1.Group("/devices")
		{
			devices.GET("", deviceHandler.List)
			devices.POST("", deviceHandler.Create)
			devices.GET("/:id", deviceHandler.Get)
			devices.PUT("/:id", deviceHandler.Update)
			devices.DELETE("/:id", deviceHandler.Delete)
			devices.POST("/:id/connect", deviceHandler.Connect)
			devices.POST("/:id/disconnect", deviceHandler.Disconnect)

			devices.GET("/:id/nodes", deviceDataHandler.BrowseNodes)
			devices.GET("/:id/data/:nodeId", deviceDataHandler.ReadData)
			devices.POST("/:id/data/:nodeId", deviceDataHandler.WriteData)
			devices.POST("/:id/subscribe", deviceDataHandler.Subscribe)
			devices.GET("/:id/subscriptions", deviceDataHandler.ListSubscriptions)
			devices.GET("/:id/subscriptions/:subId/events", deviceDataHandler.PollSubscription)
			devices.DELETE("/:id/subscriptions/:subId", deviceDataHandler.Unsubscribe)
		}

		simulators := v1.Group("/simulators")
		{
			simulators.GET("", simulatorHandler.List)
			simulators.POST("/:type/start", simulatorHandler.Start)
			simulators.POST("/:type/stop", simulatorHandler.Stop)
			simulators.PUT("/:type/config", simulatorHandler.UpdateConfig)
		}
	}

	if cfg.MCP.Enabled {
		mcp := r.Group(cfg.MCP.BasePath)
		{
			mcp.GET("/.well-known/mcp.json", mcpHandler.Discovery)
			mcp.POST("/sse", mcpHandler.SSE)
			mcp.POST("/message", mcpHandler.Message)
		}
	}

	return r
}
