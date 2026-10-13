package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AarishMansur/ShopKaro/backend/internal/models"
	"github.com/AarishMansur/ShopKaro/backend/internal/store"
)

// WeeklyStats is the input the summary is written from, so the fallback
// template and Gemma always describe the same numbers.
type WeeklyStats struct {
	Days           int                    `json:"days"`
	Revenue        float64                `json:"revenue"`
	PrevRevenue    float64                `json:"prev_revenue"`
	RevenueChange  float64                `json:"revenue_change"`
	Orders         int                    `json:"orders"`
	Units          int                    `json:"units"`
	AvgBasket      float64                `json:"avg_basket"`
	Top            []models.RankedProduct `json:"top"`
	Slow           []models.RankedProduct `json:"slow"`
	BestDay        string                 `json:"best_day"`
	BusiestHour    int                    `json:"busiest_hour"`
	NewProducts    int                    `json:"new_products"`
	ActiveProducts int                    `json:"active_products"`
}

func BuildWeeklyStats(ctx context.Context, st *store.Store, tz string) (*WeeklyStats, error) {
	summary, err := st.Summary(ctx, 7, tz)
	if err != nil {
		return nil, err
	}
	best, err := st.BestSellers(ctx, 7, 5)
	if err != nil {
		return nil, err
	}
	slow, err := st.SlowMovers(ctx, 7, 5)
	if err != nil {
		return nil, err
	}
	return &WeeklyStats{
		Days:           7,
		Revenue:        summary.Revenue,
		PrevRevenue:    summary.PrevRevenue,
		RevenueChange:  summary.RevenueChange,
		Orders:         summary.Orders,
		Units:          summary.Units,
		AvgBasket:      summary.AvgBasket,
		Top:            best,
		Slow:           slow,
		BestDay:        summary.BestDay,
		BusiestHour:    summary.BusiestHour,
		NewProducts:    summary.NewProducts,
		ActiveProducts: summary.ActiveProducts,
	}, nil
}

// WeeklySummary writes a plain-language recap in Hindi or English. Gemma writes
// it when connected; otherwise a deterministic template covers the same facts.
func WeeklySummary(ctx context.Context, o *Ollama, stats *WeeklyStats, lang string) (content, engine string, err error) {
	if lang != "hi" {
		lang = "en"
	}
	if o.Available(ctx) {
		text, err := gemmaSummary(ctx, o, stats, lang)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, "gemma", nil
		}
	}
	return templateSummary(stats, lang), "template", nil
}

func gemmaSummary(ctx context.Context, o *Ollama, stats *WeeklyStats, lang string) (string, error) {
	langWord := "English"
	if lang == "hi" {
		langWord = "Hindi (Devanagari script)"
	}
	system := fmt.Sprintf(`You write a short weekly business recap for an Indian kirana (grocery) shopkeeper.
Write in %s. Plain, warm, conversational language a shop owner reads in 20 seconds.
3 to 5 short paragraphs or bullet-style lines. Mention revenue change, busiest day/hour, top sellers, and slow movers with concrete numbers.
Use Indian number formatting (₹1,24,500). No markdown headings, no JSON, no English words mixed into the Hindi text.`, langWord)
	user := fmt.Sprintf("Last 7 days stats as JSON: %+v", stats)
	out, err := o.ChatText(ctx, o.Model(), system, user)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func templateSummary(stats *WeeklyStats, lang string) string {
	topNames := make([]string, 0, 3)
	for i, t := range stats.Top {
		if i == 3 {
			break
		}
		if t.Qty > 0 {
			topNames = append(topNames, fmt.Sprintf("%s (%d)", t.Name, t.Qty))
		}
	}
	slowNames := make([]string, 0, 3)
	for i, s := range stats.Slow {
		if i == 3 {
			break
		}
		slowNames = append(slowNames, fmt.Sprintf("%s (%d)", s.Name, s.Qty))
	}
	if len(topNames) == 0 {
		topNames = append(topNames, "—")
	}
	if len(slowNames) == 0 {
		slowNames = append(slowNames, "—")
	}
	direction := "up"
	if stats.RevenueChange < 0 {
		direction = "down"
	}
	if lang == "hi" {
		return fmt.Sprintf(`पिछले 7 दिनों का हिसाब:

इस हफ्ते कुल कमाई ₹%s हुई — पिछले हफ्ते की तुलना में %s%% %s. %d बिल बने, कुल %d सामान बिका, और औसत बिल ₹%s रहा.

सबसे ज्यादा बिकने वाला सामान: %s.
धीरे-धीरे बिकने वाला सामान: %s — इन्हें काउंटर के पास रखें तो ज्यादा बिक सकता है.

सबसे व्यस्त दिन %s रहा और सबसे ज्यादा बिक्री %d:00 बजे के आसपास हुई. नए सामान में %d आइटम जुड़े; कुल %d आइटम इस हफ्ते बिके.

हफ्ते का रिपोर्ट ShopKaro ने तैयार किया है — रीऑर्डर सूची जरूर देखें.`,
			formatINR(stats.Revenue), fmt.Sprintf("%.1f", absf(stats.RevenueChange)), directionWordHi(direction),
			stats.Orders, stats.Units, formatINR(stats.AvgBasket),
			strings.Join(topNames, ", "), strings.Join(slowNames, ", "),
			stats.BestDay, stats.BusiestHour, stats.NewProducts, stats.ActiveProducts)
	}
	return fmt.Sprintf(`Your shop earned ₹%s in the last 7 days — %s%% %s than the previous week.
You billed %d customers and sold %d units, with an average bill of ₹%s.

Best sellers: %s. Slow movers: %s — move these near the counter or run a small combo to clear them.

Your busiest day was %s and the peak hour was around %d:00. %d new products were added, and %d products sold at least once this week.

Next step: open the Reorder tab for next week's wholesale quantities.`,
		formatINR(stats.Revenue), fmt.Sprintf("%.1f", absf(stats.RevenueChange)), direction,
		stats.Orders, stats.Units, formatINR(stats.AvgBasket),
		strings.Join(topNames, ", "), strings.Join(slowNames, ", "),
		stats.BestDay, stats.BusiestHour, stats.NewProducts, stats.ActiveProducts)
}

func directionWordHi(direction string) string {
	if direction == "up" {
		return "बढ़ी"
	}
	return "घटी"
}

func absf(n float64) float64 {
	if n < 0 {
		return -n
	}
	return n
}

// formatINR renders 1234567.5 as 12,34,568 (Indian grouping).
func formatINR(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	if len(s) <= 3 {
		return s
	}
	last3 := s[len(s)-3:]
	rest := s[:len(s)-3]
	var out []string
	for len(rest) > 2 {
		out = append([]string{rest[len(rest)-2:]}, out...)
		rest = rest[:len(rest)-2]
	}
	if rest != "" {
		out = append([]string{rest}, out...)
	}
	return strings.Join(out, ",") + "," + last3
}

// Period returns the last completed week window in the shop timezone.
func Period(now time.Time) (time.Time, time.Time) {
	end := now
	start := end.AddDate(0, 0, -7)
	return start, end
}
