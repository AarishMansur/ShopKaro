package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/AarishMansur/ShopKaro/backend/internal/config"
)

const taskQueue = "shopkaro"

// Temporal runs the weekly jobs as durable schedules: they survive API
// restarts, retry flaky model calls and are inspectable in the Temporal UI.
type Temporal struct {
	Client client.Client
	Worker worker.Worker
}

// temporalDeps is captured by the registered activities.
var temporalDeps *Deps

// StartTemporal connects to a Temporal cluster, ensures both schedules exist
// and starts the worker. Cron expressions use CRON_TZ so they fire on the
// shop's local clock (e.g. Monday 07:00 IST) rather than UTC.
func StartTemporal(ctx context.Context, cfg *config.Config, deps *Deps) (*Temporal, error) {
	cli, err := client.NewClient(client.Options{
		HostPort:  cfg.TemporalAddress,
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		return nil, fmt.Errorf("temporal connect: %w", err)
	}

	temporalDeps = deps

	w := worker.New(cli, taskQueue, worker.Options{})
	w.RegisterWorkflow(WeeklySummaryWorkflow)
	w.RegisterWorkflow(ReorderForecastWorkflow)
	w.RegisterActivity(activityWeeklySummary)
	w.RegisterActivity(activityReorderForecast)
	if err := w.Start(); err != nil {
		cli.Close()
		return nil, fmt.Errorf("temporal worker: %w", err)
	}

	tz := cfg.ShopTZ.String()
	weeklyCron := fmt.Sprintf("CRON_TZ=%s 0 7 * * 1", tz)   // Mondays 07:00
	reorderCron := fmt.Sprintf("CRON_TZ=%s 0 18 * * 0", tz) // Sundays 18:00
	jobs := []struct {
		id, cron, workflow, arg string
	}{
		{"weekly-summary", weeklyCron, "WeeklySummaryWorkflow", cfg.SummaryLang},
		{"reorder-forecast", reorderCron, "ReorderForecastWorkflow", ""},
	}
	for _, j := range jobs {
		if err := ensureSchedule(ctx, cli, j.id, j.cron, j.workflow, j.arg); err != nil {
			w.Stop()
			cli.Close()
			return nil, err
		}
	}
	log.Printf("temporal: durable schedules active (summary %q, reorder %q)", weeklyCron, reorderCron)
	return &Temporal{Client: cli, Worker: w}, nil
}

func ensureSchedule(ctx context.Context, cli client.Client, id, cron, workflowName, arg string) error {
	_, err := cli.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: id,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{cron},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "shopkaro-" + id,
			Workflow:  workflowName,
			Args:      []interface{}{arg},
			TaskQueue: taskQueue,
		},
	})
	if err == nil {
		return nil
	}
	// Already exists is fine — the schedule from a previous run keeps working.
	if handle := cli.ScheduleClient().GetHandle(ctx, id); handle != nil {
		if _, dErr := handle.Describe(ctx); dErr == nil {
			log.Printf("temporal: schedule %s already exists", id)
			return nil
		}
	}
	return fmt.Errorf("temporal schedule %s: %w", id, err)
}

func WeeklySummaryWorkflow(ctx workflow.Context, lang string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	})
	return workflow.ExecuteActivity(ctx, activityWeeklySummary, lang).Get(ctx, nil)
}

func ReorderForecastWorkflow(ctx workflow.Context, _ string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	})
	return workflow.ExecuteActivity(ctx, activityReorderForecast).Get(ctx, nil)
}

func activityWeeklySummary(ctx context.Context, lang string) error {
	if temporalDeps == nil {
		return fmt.Errorf("deps not initialised")
	}
	if lang == "" {
		lang = temporalDeps.Cfg.SummaryLang
	}
	_, err := RunWeeklySummary(ctx, temporalDeps, lang)
	Mark(ctx, temporalDeps.Store, WeeklySummaryJob, err)
	return err
}

func activityReorderForecast(ctx context.Context) error {
	if temporalDeps == nil {
		return fmt.Errorf("deps not initialised")
	}
	_, err := RunReorderForecast(ctx, temporalDeps)
	Mark(ctx, temporalDeps.Store, ReorderForecastJob, err)
	return err
}
