package store

import (
	"context"
	"time"
)

type ProductSaleRow struct {
	ProductID int64     `json:"product_id"`
	SoldAt    time.Time `json:"sold_at"`
	Qty       int       `json:"qty"`
}

// ProductSalesSince returns the raw tap history used to build weekly series
// for the reorder forecast.
func (s *Store) ProductSalesSince(ctx context.Context, since time.Time) ([]ProductSaleRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT product_id, sold_at, qty FROM sales WHERE sold_at >= $1 ORDER BY sold_at`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProductSaleRow{}
	for rows.Next() {
		var r ProductSaleRow
		if err := rows.Scan(&r.ProductID, &r.SoldAt, &r.Qty); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
