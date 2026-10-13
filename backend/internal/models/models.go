package models

import "time"

type Product struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Price     float64   `json:"price"`
	MRP       float64   `json:"mrp"`
	Unit      string    `json:"unit"`
	Aliases   []string  `json:"aliases"`
	PhotoURL  *string   `json:"photo_url"`
	Emoji     string    `json:"emoji"`
	SortOrder int       `json:"sort_order"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Sale struct {
	ID        int64     `json:"id"`
	ClientID  string    `json:"client_id"`
	ProductID int64     `json:"product_id"`
	Qty       int       `json:"qty"`
	UnitPrice float64   `json:"unit_price"`
	Total     float64   `json:"total"`
	Source    string    `json:"source"`
	SoldAt    time.Time `json:"sold_at"`
	// ProductName is populated on read paths only.
	ProductName string `json:"product_name,omitempty"`
	Emoji       string `json:"emoji,omitempty"`
}

type DayPoint struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
	Qty     int     `json:"qty"`
}

type HourPoint struct {
	Hour    int     `json:"hour"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
}

type RankedProduct struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Emoji     string  `json:"emoji"`
	Category  string  `json:"category"`
	Qty       int     `json:"qty"`
	Revenue   float64 `json:"revenue"`
	Orders    int     `json:"orders"`
}

type Summary struct {
	Days            int     `json:"days"`
	Revenue         float64 `json:"revenue"`
	PrevRevenue     float64 `json:"prev_revenue"`
	RevenueChange   float64 `json:"revenue_change"`
	Orders          int     `json:"orders"`
	PrevOrders      int     `json:"prev_orders"`
	Units           int     `json:"units"`
	AvgBasket       float64 `json:"avg_basket"`
	TopProduct      string  `json:"top_product"`
	BestDay         string  `json:"best_day"`
	BusiestHour     int     `json:"busiest_hour"`
	ActiveProducts  int     `json:"active_products"`
	NewProducts     int     `json:"new_products"`
	CategoryRevenue []RankedProduct `json:"category_revenue"`
}

type ReorderRow struct {
	ProductID     int64   `json:"product_id"`
	Name          string  `json:"name"`
	Emoji         string  `json:"emoji"`
	Category      string  `json:"category"`
	Unit          string  `json:"unit"`
	Price         float64 `json:"price"`
	LastWeekQty   int     `json:"last_week_qty"`
	Avg4WeekQty   float64 `json:"avg_4week_qty"`
	ForecastQty   float64 `json:"forecast_qty"`
	SuggestedQty  int     `json:"suggested_qty"`
	Confidence    string  `json:"confidence"`
	HadHistory    bool    `json:"had_history"`
}

type Forecast struct {
	GeneratedAt time.Time    `json:"generated_at"`
	Method      string       `json:"method"` // "tabpfn" or "fallback"
	WeeksAhead  int          `json:"weeks_ahead"`
	Rows        []ReorderRow `json:"rows"`
	Note        string       `json:"note"`
}

type Report struct {
	ID          int64     `json:"id"`
	Kind        string    `json:"kind"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Lang        string    `json:"lang"`
	Content     string    `json:"content"`
	Meta        string    `json:"meta"`
	CreatedAt   time.Time `json:"created_at"`
}

type JobStatus struct {
	Mode      string            `json:"mode"` // "temporal" or "builtin"
	Enabled   bool              `json:"enabled"`
	Detail    string            `json:"detail"`
	LastRuns  map[string]JobRun `json:"last_runs"`
	NextRuns  map[string]string `json:"next_runs"`
}

type JobRun struct {
	LastRun *time.Time `json:"last_run"`
	LastOK  bool       `json:"last_ok"`
	Error   string     `json:"error,omitempty"`
}
