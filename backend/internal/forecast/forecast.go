package forecast

import (
	"context"
	"log"
	"math"
	"sort"
	"time"

	"github.com/AarishMansur/ShopKaro/backend/internal/config"
	"github.com/AarishMansur/ShopKaro/backend/internal/models"
	"github.com/AarishMansur/ShopKaro/backend/internal/store"
)

const (
	historyDays = 90
	minWeeks    = 3
	safetyPct   = 0.10
)

type Service struct {
	store  *store.Store
	cfg    *config.Config
	tabpfn *TabPFN
}

func New(st *store.Store, cfg *config.Config) *Service {
	return &Service{store: st, cfg: cfg, tabpfn: NewTabPFN(cfg.TabPFNBaseURL, cfg.TabPFNAPIKey)}
}

func (s *Service) Method() string {
	if s.tabpfn.Enabled() {
		return "tabpfn"
	}
	return "fallback"
}

type series struct {
	product       models.Product
	qty           []int // one entry per completed week, oldest first
	weekendRatio  float64
}

// Forecast predicts next week's quantity per product. TabPFN is used when a
// Prior Labs API key is configured; otherwise a seasonal-weighted fallback.
func (s *Service) Forecast(ctx context.Context, weeksAhead int) (*models.Forecast, error) {
	if weeksAhead <= 0 {
		weeksAhead = 1
	}
	products, err := s.store.ListProducts(ctx, false)
	if err != nil {
		return nil, err
	}
	sales, err := s.store.ProductSalesSince(ctx, time.Now().AddDate(0, 0, -historyDays))
	if err != nil {
		return nil, err
	}

	weeks := completedWeekStarts(s.cfg.ShopTZ, time.Now())
	if len(weeks) < minWeeks {
		return &models.Forecast{
			GeneratedAt: time.Now(),
			Method:      "fallback",
			WeeksAhead:  weeksAhead,
			Rows:        []models.ReorderRow{},
			Note:        "Not enough sales history yet — at least 3 completed weeks are needed.",
		}, nil
	}
	weekIdx := func(t time.Time) int {
		local := t.In(s.cfg.ShopTZ)
		for i := len(weeks) - 1; i >= 0; i-- {
			if !local.Before(weeks[i]) {
				return i
			}
		}
		return -1
	}

	byProduct := map[int64][]int{}
	for _, p := range products {
		byProduct[p.ID] = make([]int, len(weeks))
	}
	weekendHits := map[int64]int{}
	totalHits := map[int64]int{}
	for _, row := range sales {
		buf, ok := byProduct[row.ProductID]
		if !ok {
			continue
		}
		idx := weekIdx(row.SoldAt)
		if idx < 0 {
			continue
		}
		buf[idx] += row.Qty
		totalHits[row.ProductID]++
		dow := row.SoldAt.In(s.cfg.ShopTZ).Weekday()
		if dow == time.Saturday || dow == time.Sunday {
			weekendHits[row.ProductID]++
		}
	}

	seriesList := make([]series, 0, len(products))
	for _, p := range products {
		sr := series{product: p, qty: byProduct[p.ID]}
		if totalHits[p.ID] > 0 {
			sr.weekendRatio = float64(weekendHits[p.ID]) / float64(totalHits[p.ID])
		}
		seriesList = append(seriesList, sr)
	}

	fc := &models.Forecast{
		GeneratedAt: time.Now(),
		WeeksAhead:  weeksAhead,
		Rows:        []models.ReorderRow{},
	}

	var predictions []float64
	if s.tabpfn.Enabled() {
		predictions, err = s.tabpfn.Predict(ctx, seriesList)
		if err != nil {
			log.Printf("forecast: tabpfn unavailable, using fallback: %v", err)
		}
	}
	if predictions == nil {
		fc.Method = "fallback"
		fc.Note = "Weighted seasonal average over the last 8 weeks (set TABPFN_API_KEY for the TabPFN model)."
	} else {
		fc.Method = "tabpfn"
		fc.Note = "TabPFN-3.5 tabular foundation model trained on this shop's weekly history."
	}

	for i, sr := range seriesList {
		last, avg4, _, hadHistory := tailStats(sr.qty)
		var forecast float64
		if predictions != nil && i < len(predictions) {
			forecast = predictions[i]
		} else {
			forecast = fallbackForecast(sr.qty)
		}
		if math.IsNaN(forecast) || math.IsInf(forecast, 0) || forecast < 0 {
			forecast = 0
		}
		suggested := int(math.Ceil(forecast * (1 + safetyPct)))
		if forecast > 0 && suggested < 1 {
			suggested = 1
		}
		fc.Rows = append(fc.Rows, models.ReorderRow{
			ProductID:    sr.product.ID,
			Name:         sr.product.Name,
			Emoji:        sr.product.Emoji,
			Category:     sr.product.Category,
			Unit:         sr.product.Unit,
			Price:        sr.product.Price,
			LastWeekQty:  last,
			Avg4WeekQty:  round1(avg4),
			ForecastQty:  round1(forecast),
			SuggestedQty: suggested,
			Confidence:   confidence(sr.qty),
			HadHistory:   hadHistory,
		})
	}
	sort.Slice(fc.Rows, func(i, j int) bool {
		if fc.Rows[i].ForecastQty == fc.Rows[j].ForecastQty {
			return fc.Rows[i].Name < fc.Rows[j].Name
		}
		return fc.Rows[i].ForecastQty > fc.Rows[j].ForecastQty
	})
	return fc, nil
}

func tailStats(qty []int) (last int, avg4, avg8 float64, hadHistory bool) {
	n := len(qty)
	if n == 0 {
		return 0, 0, 0, false
	}
	last = qty[n-1]
	for _, v := range qty {
		if v > 0 {
			hadHistory = true
		}
	}
	start4 := n - 4
	if start4 < 0 {
		start4 = 0
	}
	start8 := n - 8
	if start8 < 0 {
		start8 = 0
	}
	var sum4, sum8 int
	for _, v := range qty[start4:] {
		sum4 += v
	}
	for _, v := range qty[start8:] {
		sum8 += v
	}
	avg4 = float64(sum4) / float64(n-start4)
	avg8 = float64(sum8) / float64(n-start8)
	return last, avg4, avg8, hadHistory
}

// fallbackForecast blends the most recent week, a 4-week mean and an 8-week
// mean, then nudges with the linear trend. Zero-sale weeks suppress the order
// so slow movers are not over-ordered.
func fallbackForecast(qty []int) float64 {
	if len(qty) == 0 {
		return 0
	}
	last, avg4, avg8, _ := tailStats(qty)
	recent := 0
	for _, v := range qty[len(qty)-min(4, len(qty)):] {
		recent += v
	}
	if recent == 0 {
		return 0
	}
	pred := 0.45*float64(last) + 0.35*avg4 + 0.20*avg8
	pred += slope(qty) * 0.5
	if pred < 0 {
		pred = 0
	}
	return pred
}

func slope(qty []int) float64 {
	n := len(qty)
	if n < 2 {
		return 0
	}
	start := n - 8
	if start < 0 {
		start = 0
	}
	vals := qty[start:]
	m := float64(len(vals))
	var sumX, sumY, sumXY, sumXX float64
	for i, v := range vals {
		x := float64(i)
		y := float64(v)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	denom := m*sumXX - sumX*sumX
	if denom == 0 {
		return 0
	}
	return (m*sumXY - sumX*sumY) / denom
}

func confidence(qty []int) string {
	active, n := 0, 0
	for _, v := range qty {
		n++
		if v > 0 {
			active++
		}
	}
	if n >= 6 && active*2 >= n {
		return "high"
	}
	if n >= minWeeks && active > 0 {
		return "medium"
	}
	return "low"
}

// completedWeekStarts returns Monday 00:00 in the shop timezone for every fully
// finished week, oldest first (the in-progress week is excluded).
func completedWeekStarts(loc *time.Location, now time.Time) []time.Time {
	local := now.In(loc)
	// Walk back to the most recent Monday that started before this week.
	daysBack := (int(local.Weekday()) + 6) % 7 // Monday = 0
	thisMonday := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -daysBack)
	lastCompleted := thisMonday.AddDate(0, 0, -7)

	var out []time.Time
	// 13 weeks of history is plenty for weekly features.
	for w := lastCompleted; !w.AddDate(0, 0, -7).Before(now.AddDate(0, 0, -historyDays)); w = w.AddDate(0, 0, -7) {
		out = append(out, w)
	}
	// Reverse to oldest-first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
