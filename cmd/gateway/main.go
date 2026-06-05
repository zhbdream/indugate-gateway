package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/indugate/gateway/internal/api"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/service"
	"github.com/indugate/gateway/internal/storage"
	"github.com/indugate/gateway/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	db, err := storage.NewDB(cfg.Database, log)
	if err != nil {
		log.Fatal("failed to connect database", zap.Error(err))
	}

	driverManager := service.NewDriverManager()
	simulatorService := service.NewSimulatorService(log, cfg.Simulator)
	deviceService := service.NewDeviceService(db, driverManager)

	if cfg.Bootstrap.Enabled {
		bootstrap := service.NewBootstrapService(log, db, deviceService)
		if err := bootstrap.LoadDevicesFile(context.Background(), cfg.Bootstrap.DevicesFile); err != nil {
			log.Warn("failed to bootstrap devices", zap.Error(err))
		}
	}

	authService := service.NewAuthService(db, cfg.Auth)
	if err := authService.EnsureDefaultAdmin(context.Background()); err != nil {
		log.Warn("failed to bootstrap default admin", zap.Error(err))
	}

	router, cleanup, historyService, auditService, startMetrics := api.NewRouter(cfg, log, db, api.RouterDeps{
		DriverManager:    driverManager,
		SimulatorService: simulatorService,
	})
	defer cleanup()

	metricsCtx, metricsCancel := context.WithCancel(context.Background())
	defer metricsCancel()
	startMetrics(metricsCtx)

	retentionCtx, retentionCancel := context.WithCancel(context.Background())
	historyService.StartRetentionJob(retentionCtx, log, cfg.History)
	auditService.StartRetentionJob(retentionCtx, log, cfg.Audit)

	if cfg.Auth.Enabled {
		log.Info("API authentication enabled")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		log.Info("gateway server starting",
			zap.String("addr", addr),
			zap.String("mode", cfg.Server.Mode),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server listen failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")
	retentionCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	simulatorService.Shutdown(ctx)
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown", zap.Error(err))
	}
	log.Info("server exited")
}
