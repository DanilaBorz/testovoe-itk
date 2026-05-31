package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"https://github.com/DanilaBorz/testovoe-itk/internal/config"
	"https://github.com/DanilaBorz/testovoe-itk/internal/handler"
	"https://github.com/DanilaBorz/testovoe-itk/internal/repository"
	"https://github.com/DanilaBorz/testovoe-itk/internal/service"
)

// Server представляет HTTP-сервер.
type Server struct {
	cfg    *config.Config
	router *chi.Mux
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// New создает новый Server.
func New(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.DBMaxConns)
	poolCfg.MinConns = int32(cfg.DBMinConns)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверка подключения
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("connected to database",
		zap.Int("max_conns", cfg.DBMaxConns),
		zap.Int("min_conns", cfg.DBMinConns),
	)

	// Инициализация слоев
	walletRepo := repository.NewWalletRepository(pool, logger)
	walletSvc := service.NewWalletService(walletRepo, logger)
	walletHandler := handler.NewWalletHandler(walletSvc, logger)

	// Настройка роутера
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/wallet", walletHandler.HandleOperation)
		r.Get("/wallets/{id}", walletHandler.HandleGetBalance)
	})

	return &Server{
		cfg:    cfg,
		router: router,
		pool:   pool,
		logger: logger,
	}, nil
}

// Start запускает HTTP-сервер и ожидает сигнала завершения.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.cfg.ServerPort)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s.logger.Info("server starting", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit
	s.logger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	s.pool.Close()
	s.logger.Info("server stopped gracefully")

	return nil
}