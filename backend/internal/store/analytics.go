package store

import (
	"context"
	"fmt"
	"time"

	"github.com/AarishMansur/ShopKaro/backend/internal/models"
)

func windowClause(days int) string {
	return fmt.Sprintf(`s.sold_at > now() - interval '%d days'`, days)
}

func (s *Store) Summary(ctx context.Context, days int, tz string) (*models.Summary, error) {
	if days <= 0 {
		days = 7
	}
	sum := &models.Summary{Days: days, CategoryRevenue: []models.RankedProduct{}}

	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total),0), COUNT(*), COALESCE(SUM(qty),0)
		FROM sales s WHERE `+windowClause(days)).Scan(&sum.Revenue, &sum.Orders, &sum.Units)
	if err != nil {
		return nil, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total),0), COUNT(*)
		FROM sales s WHERE s.sold_at > now() - interval '`+itoa(days*2)+` days'
		  AND s.sold_at <= now() - interval '`+itoa(days)+` days'`).Scan(&sum.PrevRevenue, &sum.PrevOrders)
	if err != nil {
		return nil, err
	}
	if sum.PrevRevenue > 0 {
		sum.RevenueChange = (sum.Revenue - sum.PrevRevenue) / sum.PrevRevenue * 100
	}
	if sum.Orders > 0 {
		sum.AvgBasket = sum.Revenue / float64(sum.Orders)
	}

	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT p.name FROM sales s JOIN products p ON p.id = s.product_id
			WHERE `+windowClause(days)+` GROUP BY p.name ORDER BY SUM(qty) DESC LIMIT 1), '—'),
		COALESCE((SELECT to_char(date(s.sold_at AT TIME ZONE $1), 'Dy, DD Mon') FROM sales s
			WHERE `+windowClause(days)+` GROUP BY date(s.sold_at AT TIME ZONE $1)
			ORDER BY SUM(total) DESC LIMIT 1), '—'),
		COALESCE((SELECT EXTRACT(HOUR FROM s.sold_at AT TIME ZONE $1)::int FROM sales s
			WHERE `+windowClause(days)+` GROUP BY EXTRACT(HOUR FROM s.sold_at AT TIME ZONE $1)
			ORDER BY SUM(total) DESC LIMIT 1), 0)`,
		tz).Scan(&sum.TopProduct, &sum.BestDay, &sum.BusiestHour)
	if err != nil {
		return nil, err
	}

	var activeProducts int
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT COUNT(DISTINCT product_id) FROM sales s WHERE `+windowClause(days)+`),
		       (SELECT COUNT(*) FROM products WHERE active AND created_at > now() - interval '`+itoa(days)+` days'),
		       (SELECT COUNT(*) FROM products WHERE active)`).
		Scan(&sum.ActiveProducts, &sum.NewProducts, &activeProducts)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT p.category, COALESCE(SUM(s.total),0), COALESCE(SUM(s.qty),0), COUNT(*)
		FROM sales s JOIN products p ON p.id = s.product_id
		WHERE `+windowClause(days)+`
		GROUP BY p.category ORDER BY SUM(s.total) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r models.RankedProduct
		if err := rows.Scan(&r.Category, &r.Revenue, &r.Qty, &r.Orders); err != nil {
			return nil, err
		}
		r.Name = r.Category
		sum.CategoryRevenue = append(sum.CategoryRevenue, r)
	}
	return sum, rows.Err()
}

func (s *Store) BestSellers(ctx context.Context, days, limit int) ([]models.RankedProduct, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.name, p.emoji, p.category, COALESCE(SUM(s.qty),0), COALESCE(SUM(s.total),0), COUNT(s.id)
		FROM sales s JOIN products p ON p.id = s.product_id
		WHERE `+windowClause(days)+`
		GROUP BY p.id, p.name, p.emoji, p.category
		ORDER BY SUM(s.qty) DESC, SUM(s.total) DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.RankedProduct{}
	for rows.Next() {
		var r models.RankedProduct
		if err := rows.Scan(&r.ProductID, &r.Name, &r.Emoji, &r.Category, &r.Qty, &r.Revenue, &r.Orders); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SlowMovers ranks active products by units sold in the window, weakest first,
// including products with no sales at all.
func (s *Store) SlowMovers(ctx context.Context, days, limit int) ([]models.RankedProduct, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.name, p.emoji, p.category, COALESCE(SUM(s.qty),0), COALESCE(SUM(s.total),0), COUNT(s.id)
		FROM products p
		LEFT JOIN sales s ON s.product_id = p.id AND `+windowClause(days)+`
		WHERE p.active
		GROUP BY p.id, p.name, p.emoji, p.category
		ORDER BY SUM(s.qty) ASC, p.name ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.RankedProduct{}
	for rows.Next() {
		var r models.RankedProduct
		if err := rows.Scan(&r.ProductID, &r.Name, &r.Emoji, &r.Category, &r.Qty, &r.Revenue, &r.Orders); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) ByDay(ctx context.Context, days int, tz string) ([]models.DayPoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(date(s.sold_at AT TIME ZONE $1), 'YYYY-MM-DD'),
		       COALESCE(SUM(s.total),0), COUNT(*), COALESCE(SUM(s.qty),0)
		FROM sales s
		WHERE s.sold_at > now() - interval '`+itoa(days)+` days'
		GROUP BY 1 ORDER BY 1`, tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byDate := map[string]models.DayPoint{}
	for rows.Next() {
		var d models.DayPoint
		if err := rows.Scan(&d.Date, &d.Revenue, &d.Orders, &d.Qty); err != nil {
			return nil, err
		}
		byDate[d.Date] = d
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	out := make([]models.DayPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		label := time.Now().In(loc).AddDate(0, 0, -i).Format("2006-01-02")
		if d, ok := byDate[label]; ok {
			out = append(out, d)
		} else {
			out = append(out, models.DayPoint{Date: label})
		}
	}
	return out, nil
}

func (s *Store) ByHour(ctx context.Context, days int, tz string) ([]models.HourPoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT EXTRACT(HOUR FROM s.sold_at AT TIME ZONE $1)::int, COALESCE(SUM(s.total),0), COUNT(*)
		FROM sales s
		WHERE `+windowClause(days)+`
		GROUP BY 1 ORDER BY 1`, tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.HourPoint, 24)
	for h := 0; h < 24; h++ {
		out[h] = models.HourPoint{Hour: h}
	}
	for rows.Next() {
		var h int
		var hp models.HourPoint
		if err := rows.Scan(&h, &hp.Revenue, &hp.Orders); err != nil {
			return nil, err
		}
		if h >= 0 && h < 24 {
			hp.Hour = h
			out[h] = hp
		}
	}
	return out, rows.Err()
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
