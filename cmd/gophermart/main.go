package main

import (
	"log"
	"net/http"
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

	jwtSecret := []byte("supersecretjwt") //TODO: change??
	router := api.InitRouter(db, jwtSecret)

	go accrual.Worker(db, cfg.AccrualSystemAddress, 3*time.Second)

	logger.Infof("server starting at %s", cfg.RunAddress)

	if err := http.ListenAndServe(cfg.RunAddress, router); err != nil {
		logger.Fatalf("cannot start server: %v", err)
	}
}
