package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antonminaichev/gophermart-loyalty/internal/balance"
	"github.com/antonminaichev/gophermart-loyalty/internal/logger"
	"github.com/antonminaichev/gophermart-loyalty/internal/order"
	"github.com/antonminaichev/gophermart-loyalty/internal/router"
	storage "github.com/antonminaichev/gophermart-loyalty/internal/storage/postgres"
	"github.com/antonminaichev/gophermart-loyalty/internal/user"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store, err := storage.NewPostgresStorage(cfg.DatabaseConnection)
	if err != nil {
		log.Fatalf("Failed to initialize Postgres storage: %v", err)
	}
	if err := store.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Warning: failed to close storage: %v", err)
		}
	}()

	userSvc := user.NewService(store, []byte(cfg.JWTSecret), cfg.JWTTTL)
	userHandler := user.NewHandler(userSvc)

	orderSvc := order.NewService(store)
	orderHandler := order.NewHandler(orderSvc)

	balanceSvc := balance.NewService(store, store)
	balanceHandler := balance.NewHandler(balanceSvc)

	r := router.NewRouter(userHandler, orderHandler, balanceHandler, []byte(cfg.JWTSecret), store)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	accrual := &order.HTTPAccrualClient{
		Client:         httpClient,
		AccrualAddress: cfg.AccrualAdress,
	}

	go func() {
		order.DispatcherLoop(
			ctx,
			accrual,
			orderSvc,
			cfg.AccrualWorkers,
			cfg.AccrualInterval,
		)
	}()

	go func() {
		log.Printf("Starting server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}
