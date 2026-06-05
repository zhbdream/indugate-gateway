package api

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/handler"
	"github.com/indugate/gateway/internal/api/middleware"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/metrics"
	"github.com/indugate/gateway/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RouterDeps struct {
	DriverManager    *service.DriverManager
	SimulatorService *service.SimulatorService
}

func NewRouter(cfg *config.Config, log *zap.Logger, db *gorm.DB, deps RouterDeps) (*gin.Engine, func(), *service.HistoryService, *service.AuditService, func(context.Context)) {
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logger(log))
	r.Use(middleware.CORS())

	authService := service.NewAuthService(db, cfg.Auth)
	devicePermService := service.NewDevicePermissionService(db, cfg.Auth)
	auditService := service.NewAuditService(db)
	r.Use(middleware.Auth(cfg.Auth, authService))
	r.Use(middleware.RBAC(cfg.Auth))
	r.Use(middleware.Audit(auditService, cfg.Audit, cfg.Auth))
	if cfg.Metrics.Enabled {
		metrics.Register()
		r.Use(middleware.Prometheus())
	}

	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(authService, devicePermService)
	deviceService := service.NewDeviceService(db, deps.DriverManager)
	historyService := service.NewHistoryService(db)
	alertNotifier := service.NewAlertNotifier(log, cfg.Alerts)
	alertService := service.NewAlertService(db, alertNotifier)
	influxWriter := service.NewInfluxWriter(log, cfg.InfluxDB)
	recorder := service.NewHistoryRecorder(historyService, alertService, influxWriter)

	deviceHandler := handler.NewDeviceHandler(deviceService, devicePermService)
	deviceDataHandler := handler.NewDeviceDataHandler(deviceService, recorder, devicePermService)
	historyHandler := handler.NewHistoryHandler(historyService, deviceService, devicePermService)
	alertHandler := handler.NewAlertHandler(alertService, devicePermService)
	dashboardHandler := handler.NewDashboardHandler(alertService, devicePermService)
	auditHandler := handler.NewAuditHandler(auditService)
	simulatorHandler := handler.NewSimulatorHandler(deps.SimulatorService)
	mcpHandler := handler.NewMCPHandler(cfg.MCP, deviceService, devicePermService)
	swaggerHandler := handler.NewSwaggerHandler()

	r.GET("/health", healthHandler.Check)
	if cfg.Metrics.Enabled {
		r.GET("/metrics", handler.Metrics)
	}

	v1 := r.Group("/api/v1")
	{
		v1.GET("/auth/config", authHandler.Config)
		v1.POST("/auth/login", authHandler.Login)
		v1.GET("/auth/me", authHandler.Me)

		users := v1.Group("/users")
		{
			users.GET("", userHandler.List)
			users.POST("", userHandler.Create)
			users.PUT("/:id", userHandler.Update)
			users.PUT("/:id/password", userHandler.ChangePassword)
			users.DELETE("/:id", userHandler.Delete)
			users.GET("/:id/devices", userHandler.ListDevices)
			users.PUT("/:id/devices", userHandler.SetDevices)
		}

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
			devices.GET("/:id/data/history", historyHandler.QueryHistory)
			devices.GET("/:id/data/history/export", historyHandler.ExportCSV)
			devices.GET("/:id/data/*nodeId", deviceDataHandler.ReadData)
			devices.POST("/:id/data/*nodeId", deviceDataHandler.WriteData)
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

		alerts := v1.Group("/alerts")
		{
			alerts.GET("/rules", alertHandler.ListRules)
			alerts.POST("/rules", alertHandler.CreateRule)
			alerts.PUT("/rules/:id", alertHandler.UpdateRule)
			alerts.DELETE("/rules/:id", alertHandler.DeleteRule)
			alerts.GET("/events", alertHandler.ListEvents)
			alerts.POST("/events/:id/acknowledge", alertHandler.AcknowledgeEvent)
		}

		v1.GET("/dashboard/stats", dashboardHandler.Stats)

		v1.GET("/audit/logs", auditHandler.List)
	}

	r.GET("/swagger/index.html", swaggerHandler.Index)
	r.GET("/swagger/openapi.json", swaggerHandler.OpenAPI)

	if cfg.MCP.Enabled {
		mcpGroup := r.Group(cfg.MCP.BasePath)
		{
			mcpGroup.GET("/.well-known/mcp.json", mcpHandler.Discovery)
			mcpGroup.GET("/sse", mcpHandler.SSEStream)
			mcpGroup.POST("/sse", mcpHandler.Message)
			mcpGroup.POST("/message", mcpHandler.Message)
		}
	}

	registerWebStatic(r, cfg.Web.StaticDir, log)

	cleanup := func() {
		if influxWriter != nil {
			influxWriter.Close()
		}
		if alertNotifier != nil {
			alertNotifier.Close()
		}
	}
	startMetrics := func(ctx context.Context) {
		if cfg.Metrics.Enabled {
			metrics.StartRefresh(ctx, db, 15*time.Second)
		}
	}
	return r, cleanup, historyService, auditService, startMetrics
}

func registerWebStatic(r *gin.Engine, staticDir string, log *zap.Logger) {
	if staticDir == "" {
		staticDir = "./web/dist"
	}

	info, err := os.Stat(staticDir)
	if err != nil || !info.IsDir() {
		log.Warn("web static directory not found, UI disabled", zap.String("dir", staticDir))
		return
	}

	indexFile := strings.TrimRight(staticDir, "/\\") + "/index.html"
	r.Static("/assets", staticDir+"/assets")
	r.StaticFile("/favicon.svg", staticDir+"/favicon.svg")

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/mcp") || path == "/health" || strings.HasPrefix(path, "/swagger") {
			response.NotFound(c, "not found")
			return
		}
		c.File(indexFile)
	})

	log.Info("web UI enabled", zap.String("static_dir", staticDir))
}
