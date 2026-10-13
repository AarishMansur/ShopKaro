package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AarishMansur/ShopKaro/backend/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

const productCols = `id, name, category, price, mrp, unit, aliases, photo_url, emoji, sort_order, active, created_at, updated_at`

func scanProduct(row pgx.Row) (*models.Product, error) {
	var p models.Product
	var aliases []byte
	err := row.Scan(&p.ID, &p.Name, &p.Category, &p.Price, &p.MRP, &p.Unit, &aliases, &p.PhotoURL,
		&p.Emoji, &p.SortOrder, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Aliases = []string{}
	if len(aliases) > 0 {
		_ = json.Unmarshal(aliases, &p.Aliases)
	}
	return &p, nil
}

func (s *Store) ListProducts(ctx context.Context, includeInactive bool) ([]models.Product, error) {
	q := fmt.Sprintf(`SELECT %s FROM products`, productCols)
	if !includeInactive {
		q += ` WHERE active = TRUE`
	}
	q += ` ORDER BY sort_order, name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *Store) GetProduct(ctx context.Context, id int64) (*models.Product, error) {
	return scanProduct(s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM products WHERE id = $1`, productCols), id))
}

type ProductInput struct {
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Price    float64  `json:"price"`
	MRP      float64  `json:"mrp"`
	Unit     string   `json:"unit"`
	Aliases  []string `json:"aliases"`
	PhotoURL *string  `json:"photo_url"`
	Emoji    string   `json:"emoji"`
	SortOrder int     `json:"sort_order"`
	Active   *bool    `json:"active"`
}

func (in *ProductInput) normalize() {
	if in.Category == "" {
		in.Category = "General"
	}
	if in.Unit == "" {
		in.Unit = "pcs"
	}
	if in.MRP == 0 {
		in.MRP = in.Price
	}
	if in.Aliases == nil {
		in.Aliases = []string{}
	}
}

func (s *Store) CreateProduct(ctx context.Context, in ProductInput) (*models.Product, error) {
	in.normalize()
	aliases, _ := json.Marshal(in.Aliases)
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO products (name, category, price, mrp, unit, aliases, photo_url, emoji, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		in.Name, in.Category, in.Price, in.MRP, in.Unit, aliases, in.PhotoURL, in.Emoji, in.SortOrder,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, id)
}

func (s *Store) UpdateProduct(ctx context.Context, id int64, in ProductInput) (*models.Product, error) {
	in.normalize()
	aliases, _ := json.Marshal(in.Aliases)
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE products SET name = $1, category = $2, price = $3, mrp = $4, unit = $5,
			aliases = $6, photo_url = $7, emoji = $8, sort_order = $9, active = $10, updated_at = now()
		WHERE id = $11`,
		in.Name, in.Category, in.Price, in.MRP, in.Unit, aliases, in.PhotoURL, in.Emoji, in.SortOrder, active, id,
	)
	if err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, id)
}

// DeleteProduct removes the product unless it has sales history, in which case
// it is archived instead so old receipts still resolve.
func (s *Store) DeleteProduct(ctx context.Context, id int64) (string, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM products WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM sales WHERE product_id = $1)`, id)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() > 0 {
		return "deleted", nil
	}
	tag, err = s.pool.Exec(ctx, `UPDATE products SET active = FALSE, updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", pgx.ErrNoRows
	}
	return "archived", nil
}
