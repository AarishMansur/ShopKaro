package jobs

import (
	"context"
	"log"
	"time"
)

// Scheduler is the dependency-free fallback used when Temporal is disabled.
// Due-ness is derived from wall clock, and the last run is persisted in the
// database, so restarts never skip or double-fire a report.
type Scheduler struct {
	deps     *Deps
	interval time.Duration
}

func NewScheduler(deps *Deps) *Scheduler {
	return &Scheduler{deps: deps, interval: 5 * time.Minute}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.checkOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkOnce(ctx)
		}
	}
}

func (s *Scheduler) checkOnce(ctx context.Context) {
	now := time.Now()
	loc := s.deps.Cfg.ShopTZ

	checks := []struct {
		name string
		due  time.Time
		run  func(context.Context) error
	}{
		{
			name: WeeklySummaryJob,
			due:  MostRecentDue(WeeklySummaryJob, now, loc),
			run: func(ctx context.Context) error {
				_, err := RunWeeklySummary(ctx, s.deps, s.deps.Cfg.SummaryLang)
				return err
			},
		},
		{
			name: ReorderForecastJob,
			due:  MostRecentDue(ReorderForecastJob, now, loc),
			run: func(ctx context.Context) error {
				_, err := RunReorderForecast(ctx, s.deps)
				return err
			},
		},
	}

	for _, c := range checks {
		run, err := s.deps.Store.JobRun(ctx, c.name)
		if err != nil {
			log.Printf("scheduler: read job state %s: %v", c.name, err)
			continue
		}
		if run.LastRun != nil && !run.LastRun.Before(c.due) {
			continue
		}
		if run.LastRun != nil && now.Sub(*run.LastRun) < time.Minute {
			continue
		}
		log.Printf("scheduler: running %s", c.name)
		runErr := c.run(ctx)
		Mark(ctx, s.deps.Store, c.name, runErr)
		if runErr != nil {
			log.Printf("scheduler: %s failed: %v", c.name, runErr)
		}
	}
}

// MostRecentDue returns the last time the named job should have fired.
func MostRecentDue(name string, now time.Time, loc *time.Location) time.Time {
	local := now.In(loc)
	switch name {
	case WeeklySummaryJob:
		// Mondays 07:00.
		daysBack := (int(local.Weekday()) + 6) % 7
		thisMonday := at(local, time.Monday, daysBack, 7)
		if !local.Before(thisMonday) {
			return thisMonday
		}
		return thisMonday.AddDate(0, 0, -7)
	case ReorderForecastJob:
		// Sundays 18:00.
		daysBack := int(local.Weekday()) // Sunday = 0
		thisSunday := at(local, time.Sunday, daysBack, 18)
		if !local.Before(thisSunday) {
			return thisSunday
		}
		return thisSunday.AddDate(0, 0, -7)
	}
	return time.Time{}
}

func at(base time.Time, weekday time.Weekday, daysBack, hour int) time.Time {
	day := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).AddDate(0, 0, -daysBack)
	return time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, base.Location())
}

// NextDue returns the upcoming fire time for status display.
func NextDue(name string, now time.Time, loc *time.Location) time.Time {
	last := MostRecentDue(name, now, loc)
	return last.AddDate(0, 0, 7)
}
