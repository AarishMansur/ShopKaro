package forecast

import (
	"testing"
	"time"
)

func TestFallbackForecastSteadyDemand(t *testing.T) {
	got := fallbackForecast([]int{5, 5, 5, 5, 5, 5, 5, 5})
	if got < 4.5 || got > 5.5 {
		t.Errorf("steady series forecast = %.2f, want ~5", got)
	}
}

func TestFallbackForecastZeroRecentWeeks(t *testing.T) {
	if got := fallbackForecast([]int{4, 4, 4, 4, 0, 0, 0, 0}); got != 0 {
		t.Errorf("forecast with no recent sales = %.2f, want 0", got)
	}
}

func TestFallbackForecastGrowth(t *testing.T) {
	flat := fallbackForecast([]int{3, 3, 3, 3, 3, 3, 3, 3})
	growing := fallbackForecast([]int{1, 2, 3, 4, 5, 6, 7, 8})
	if growing <= flat {
		t.Errorf("growing series (%.2f) should forecast above flat (%.2f)", growing, flat)
	}
}

func TestConfidenceLevels(t *testing.T) {
	if c := confidence([]int{4, 5, 4, 6, 5, 7, 4, 6}); c != "high" {
		t.Errorf("dense history confidence = %s, want high", c)
	}
	if c := confidence([]int{0, 0, 0, 0}); c != "low" {
		t.Errorf("no history confidence = %s, want low", c)
	}
}

func TestCompletedWeekStartsAreMondaysOldestFirst(t *testing.T) {
	loc := time.FixedZone("IST", 5*60*60+30*60)
	now := time.Date(2026, 10, 14, 15, 0, 0, 0, loc) // Wednesday
	weeks := completedWeekStarts(loc, now)
	if len(weeks) < minWeeks {
		t.Fatalf("expected at least %d weeks, got %d", minWeeks, len(weeks))
	}
	for i, w := range weeks {
		if w.Weekday() != time.Monday {
			t.Errorf("week %d starts on %s, want Monday", i, w.Weekday())
		}
		if i > 0 && !weeks[i].After(weeks[i-1]) {
			t.Errorf("weeks are not ascending at %d", i)
		}
	}
	last := weeks[len(weeks)-1]
	if !last.Before(now) {
		t.Errorf("in-progress week must be excluded, last week = %v", last)
	}
}

func TestSlope(t *testing.T) {
	if s := slope([]int{1, 2, 3, 4, 5}); s < 0.9 {
		t.Errorf("slope of linear growth = %.2f, want >= 0.9", s)
	}
	if s := slope([]int{5, 5, 5}); s != 0 {
		t.Errorf("slope of flat series = %.2f, want 0", s)
	}
}
