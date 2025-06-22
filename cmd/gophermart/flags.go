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
	ReadTimeout        time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout       time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	IdleTimeout        time.Duration `env:"READ_TIMEOUT" envDefault:"120s"`
	ClientTimeout      time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	MigrationDir       string        `env:"MIGRATION_DIRECTORY" envDefault:"./migrations"`
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
	readTimeout := flag.Duration("rt", cfg.ReadTimeout, "Server read timeout")
	writeTimeout := flag.Duration("wt", cfg.WriteTimeout, "Server write timeout")
	idleTimeout := flag.Duration("it", cfg.IdleTimeout, "Server idle timeout")
	clientTimeout := flag.Duration("ct", cfg.ClientTimeout, "Client timeout")
	migrationDir := flag.String("md", cfg.MigrationDir, "Directory with DB migrations")
	databaseConnection := flag.String("d", cfg.DatabaseConnection, "Database connection string")
	jwtTTL := flag.Duration("t", cfg.JWTTTL, "TTL for JWT token(e.g. 24h; 30m )")

	flag.Parse()

	cfg.Address = *address
	cfg.LogLevel = *loglevel
	cfg.AccrualAdress = *accrualAddress
	cfg.AccrualWorkers = *accrualWorkers
	cfg.AccrualInterval = *accrualInterval
	cfg.ReadTimeout = *readTimeout
	cfg.WriteTimeout = *writeTimeout
	cfg.IdleTimeout = *idleTimeout
	cfg.ClientTimeout = *clientTimeout
	cfg.MigrationDir = *migrationDir
	cfg.DatabaseConnection = *databaseConnection
	cfg.JWTTTL = *jwtTTL

	return cfg, nil
}
