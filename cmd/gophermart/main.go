package main

import (
	"log"
	"net/http"

	"github.com/JinFuuMugen/gophermart-ya/config"
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

	jwtSecret := []byte("supersecretjwt")
	router := api.InitRouter(db, jwtSecret)

	logger.Infof("server starting at %s", cfg.RunAddress)

	if err := http.ListenAndServe(cfg.RunAddress, router); err != nil {
		logger.Fatalf("cannot start server: %v", err)
	}
}
