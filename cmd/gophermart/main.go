package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JinFuuMugen/gophermart-ya/config"
	"github.com/JinFuuMugen/gophermart-ya/internal/accrual"
	"github.com/JinFuuMugen/gophermart-ya/internal/api"
	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
	"github.com/JinFuuMugen/gophermart-ya/internal/storage"
)

func main() {
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("cannot init custom logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.LoadGophermartConfig()
	if err != nil {
		logger.Fatalf("cannot load config: %v", err)
	}

	db, err := storage.New(cfg.DatabaseURI)
	if err != nil {
		logger.Fatalf("cannot init database: %v", err)
	}

	if err := storage.RunMigrations(cfg.DatabaseURI); err != nil {
		logger.Fatalf("cannot migrate database: %v", err)
	}

	jwtSecret := []byte("supersecretjwt")
	router := api.InitRouter(db, jwtSecret)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go accrual.Worker(rootCtx, db, cfg.AccrualSystemAddress, 3*time.Second)

	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	go func() {
		logger.Infof("server starting at %s", cfg.RunAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("cannot start server: %v", err)
		}
	}()

	<-rootCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("server shutdown error: %v", err)
	}

	logger.Infof("server stopped gracefully")
}
