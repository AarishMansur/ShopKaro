package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/AarishMansur/ShopKaro/backend/internal/models"
)

type SaleInput struct {
	ClientID  string     `json:"client_id"`
	ProductID int64      `json:"product_id"`
	Qty       int        `json:"qty"`
	UnitPrice *float64   `json:"unit_price"`
	SoldAt    *time.Time `json:"sold_at"`
	Source    string     `json:"source"`
}

const saleCols = `s.id, s.client_id, s.product_id, s.qty, s.unit_price, s.total, s.source, s.sold_at, p.name, p.emoji`

func scanSale(row pgx.Row) (*models.Sale, error) {
	var sl models.Sale
	err := row.Scan(&sl.ID, &sl.ClientID, &sl.ProductID, &sl.Qty, &sl.UnitPrice, &sl.Total, &sl.Source, &sl.SoldAt, &sl.ProductName, &sl.Emoji)
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// InsertSales is idempotent on client_id so an offline device can replay its
// queue without double counting.
func (s *Store) InsertSales(ctx context.Context, inputs []SaleInput) ([]models.Sale, error) {
	out := make([]models.Sale, 0, len(inputs))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	for _, in := range inputs {
		if in.Qty <= 0 || in.ClientID == "" {
			continue
		}
		var existingID int64
		err := tx.QueryRow(ctx, `SELECT id FROM sales WHERE client_id = $1`, in.ClientID).Scan(&existingID)
		if err == nil {
			sale, err := scanSale(tx.QueryRow(ctx, `
				SELECT `+saleCols+` FROM sales s JOIN products p ON p.id = s.product_id WHERE s.id = $1`, existingID))
			if err == nil {
				out = append(out, *sale)
			}
			continue
		}
		if err != pgx.ErrNoRows {
			return nil, err
		}

		price := 0.0
		if in.UnitPrice != nil {
			price = *in.UnitPrice
		} else {
			if err := tx.QueryRow(ctx, `SELECT price FROM products WHERE id = $1 AND active = TRUE`, in.ProductID).Scan(&price); err != nil {
				if err == pgx.ErrNoRows {
					continue
				}
				return nil, err
			}
		}
		soldAt := time.Now().UTC()
		if in.SoldAt != nil {
			soldAt = in.SoldAt.UTC()
		}
		source := in.Source
		if source == "" {
			source = "tap"
		}
		var id int64
		err = tx.QueryRow(ctx, `
			INSERT INTO sales (client_id, product_id, qty, unit_price, total, source, sold_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			in.ClientID, in.ProductID, in.Qty, price, price*float64(in.Qty), source, soldAt,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("insert sale: %w", err)
		}
		sale, err := scanSale(tx.QueryRow(ctx, `
			SELECT `+saleCols+` FROM sales s JOIN products p ON p.id = s.product_id WHERE s.id = $1`, id))
		if err != nil {
			return nil, err
		}
		out = append(out, *sale)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListSales(ctx context.Context, limit int) ([]models.Sale, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+saleCols+` FROM sales s JOIN products p ON p.id = s.product_id
		ORDER BY s.sold_at DESC, s.id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Sale{}
	for rows.Next() {
		sale, err := scanSale(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sale)
	}
	return out, rows.Err()
}

func (s *Store) UpdateSaleQty(ctx context.Context, id int64, qty int) (*models.Sale, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("qty must be positive")
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE sales SET qty = $1, total = unit_price * $1
		WHERE id = $2 AND qty <> $1`, qty, id)
	if err != nil {
		return nil, err
	}
	return scanSale(s.pool.QueryRow(ctx, `
		SELECT `+saleCols+` FROM sales s JOIN products p ON p.id = s.product_id WHERE s.id = $1`, id))
}

func (s *Store) DeleteSale(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sales WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
