package jobs

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/AarishMansur/ShopKaro/backend/internal/ai"
	"github.com/AarishMansur/ShopKaro/backend/internal/config"
	"github.com/AarishMansur/ShopKaro/backend/internal/forecast"
	"github.com/AarishMansur/ShopKaro/backend/internal/models"
	"github.com/AarishMansur/ShopKaro/backend/internal/store"
)

const (
	WeeklySummaryJob = "weekly_summary"
	ReorderForecastJob = "reorder_forecast"
)

type Deps struct {
	Cfg      *config.Config
	Store    *store.Store
	Forecast *forecast.Service
	AI       *ai.Ollama
}

// RunWeeklySummary writes the plain-language recap and persists it so the
// frontend can show the latest scheduled report.
func RunWeeklySummary(ctx context.Context, d *Deps, lang string) (*models.Report, error) {
	stats, err := ai.BuildWeeklyStats(ctx, d.Store, d.Cfg.ShopTZ.String())
	if err != nil {
		return nil, err
	}
	content, engine, err := ai.WeeklySummary(ctx, d.AI, stats, lang)
	if err != nil {
		return nil, err
	}
	meta, _ := json.Marshal(map[string]any{
		"engine":  engine,
		"revenue": stats.Revenue,
		"orders":  stats.Orders,
		"change":  stats.RevenueChange,
	})
	start, end := ai.Period(time.Now())
	return d.Store.SaveReport(ctx, WeeklySummaryJob, lang, start, end, content, string(meta))
}

// RunReorderForecast persists next week's quantities from a fresh forecast run.
func RunReorderForecast(ctx context.Context, d *Deps) (*models.Report, error) {
	fc, err := d.Forecast.Forecast(ctx, 1)
	if err != nil {
		return nil, err
	}
	content, err := json.Marshal(fc)
	if err != nil {
		return nil, err
	}
	meta, _ := json.Marshal(map[string]any{"method": fc.Method, "rows": len(fc.Rows)})
	start, end := ai.Period(time.Now())
	return d.Store.SaveReport(ctx, ReorderForecastJob, "en", start, end, string(content), string(meta))
}

// Mark records a job attempt in the job_runs table (shared by both schedulers).
func Mark(ctx context.Context, st *store.Store, name string, err error) {
	if recErr := st.RecordJobRun(ctx, name, err); recErr != nil {
		log.Printf("jobs: record run %s: %v", name, recErr)
	}
}
