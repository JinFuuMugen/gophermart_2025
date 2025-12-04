package config

import (
	"errors"
	"flag"
	"os"
)

type GophermartConfig struct {
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	RunAddress           string `env:"RUN_ADDRESS"`
}

func LoadGophermartConfig() (*GophermartConfig, error) {
	cfg := &GophermartConfig{
		AccrualSystemAddress: "",
		DatabaseURI:          "",
		RunAddress:           "",
	}

	flag.StringVar(&cfg.AccrualSystemAddress, "r", cfg.AccrualSystemAddress, "accrual system address")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "database uri")
	flag.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "gophermart address")
	flag.Parse()

	if envAccrualSystemAddress, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		cfg.AccrualSystemAddress = envAccrualSystemAddress
	}

	if envDatabaseURI, ok := os.LookupEnv("DATABASE_URI"); ok {
		cfg.DatabaseURI = envDatabaseURI
	}

	if envRunAddress, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		cfg.RunAddress = envRunAddress
	}

	if cfg.AccrualSystemAddress == "" {
		return nil, errors.New("accrual system address is empty")
	}

	if cfg.DatabaseURI == "" {
		return nil, errors.New("database URI is empty")
	}

	if cfg.RunAddress == "" {
		return nil, errors.New("run address  is empty")
	}

	return cfg, nil
}
