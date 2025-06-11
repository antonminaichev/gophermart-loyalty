package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/caarlos0/env"
)

type Config struct {
	Address            string        `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	LogLevel           string        `env:"LOG_LEVEL" envDefault:"INFO"`
	AccrualAdress      string        `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"localhost:8090"`
	AccrualWorkers     int           `env:"ACCRUAL_WORKER" envDefault:"30"`
	AccrualInterval    time.Duration `env:"ACCRUAL_INTERVAL" envDefault:"50ms"`
	DatabaseConnection string        `env:"DATABASE_URI"`
	JWTSecret          string        `env:"JWT_SECRET" envDefault:"dontexposethis"`
	JWTTTL             time.Duration `env:"JWT_TTL" envDefault:"24h"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("ENV JWT_SECRET must be set")
	}

	address := flag.String("a", cfg.Address, "{Host:port} for server")
	loglevel := flag.String("l", cfg.LogLevel, "Log level for server")
	accrualAddress := flag.String("r", cfg.AccrualAdress, "File storage path")
	accrualWorkers := flag.Int("w", cfg.AccrualWorkers, "Size of worker pool")
	accrualInterval := flag.Duration("i", cfg.AccrualInterval, "Worker pull interval")
	databaseConnection := flag.String("d", cfg.DatabaseConnection, "Database connection string")
	jwtTTL := flag.Duration("t", cfg.JWTTTL, "TTL for JWT token(e.g. 24h; 30m )")

	flag.Parse()

	cfg.Address = *address
	cfg.LogLevel = *loglevel
	cfg.AccrualAdress = *accrualAddress
	cfg.AccrualWorkers = *accrualWorkers
	cfg.AccrualInterval = *accrualInterval
	cfg.DatabaseConnection = *databaseConnection
	cfg.JWTTTL = *jwtTTL

	return cfg, nil
}
