package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AarishMansur/ShopKaro/backend/internal/ai"
	"github.com/AarishMansur/ShopKaro/backend/internal/config"
	"github.com/AarishMansur/ShopKaro/backend/internal/db"
	"github.com/AarishMansur/ShopKaro/backend/internal/forecast"
	"github.com/AarishMansur/ShopKaro/backend/internal/httpapi"
	"github.com/AarishMansur/ShopKaro/backend/internal/jobs"
	"github.com/AarishMansur/ShopKaro/backend/internal/store"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	st := store.New(pool)
	ollama := ai.NewOllama(cfg.OllamaURL, cfg.GemmaModel, cfg.GemmaVisionModel)
	fc := forecast.New(st, cfg)
	deps := &jobs.Deps{Cfg: cfg, Store: st, Forecast: fc, AI: ollama}

	api := httpapi.New(cfg, st, ollama, fc, deps)

	// Temporal is optional: when enabled it owns the schedules, otherwise the
	// built-in scheduler runs the identical jobs.
	if cfg.TemporalEnabled {
		t, err := jobs.StartTemporal(ctx, cfg, deps)
		if err != nil {
			log.Printf("temporal: unavailable (%v), falling back to built-in scheduler", err)
			api.SetTemporal(false, "temporal-error")
			go jobs.NewScheduler(deps).Run(ctx)
		} else {
			api.SetTemporal(true, "")
			defer func() {
				t.Worker.Stop()
				t.Client.Close()
			}()
		}
	} else {
		go jobs.NewScheduler(deps).Run(ctx)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("ShopKaro API listening on :%s (shop=%s tz=%s)", cfg.Port, cfg.ShopName, cfg.ShopTZ)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http: %v", err)
	}
}
