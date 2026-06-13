package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"org-tree-api/internal/api"
	"org-tree-api/internal/config"
	"org-tree-api/internal/lib/logger"
	"org-tree-api/internal/service"
	"org-tree-api/pkg/api/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// main godoc
//
//	@title			API организационной структуры
//	@version		1.0
//	@license.name	Apache 2.0
//	@basepath		/
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		slog.Error("abnormal shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server shutdown successfully")
}

func run(ctx context.Context, cfg config.Config) (err error) {
	slog.SetDefault(logger.New(cfg.Logger))

	storage, err := openStorage(cfg)
	if err != nil {
		return err
	}
	if storage, ok := storage.(io.Closer); ok {
		defer func() {
			if closeErr := storage.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}()
	}

	api := api.New(service.New(storage))
	router := http.NewServeMux()
	router.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
	router.Handle(docs.SwaggerInfo.BasePath, logger.HTTPLogging(slog.Default(), api))

	server := http.Server{
		Handler:      requestTimeout(cfg.Server.RequestTimeout, router),
		Addr:         cfg.Server.Addr,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	done := make(chan error, 1)
	go func() {
		defer close(done)
		slog.Info("startup server", "addr", server.Addr)
		done <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown server", "cause", context.Cause(ctx))
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-done:
		return err
	}
}

func requestTimeout(d time.Duration, h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		h.ServeHTTP(w, r.WithContext(ctx))
	}
}
