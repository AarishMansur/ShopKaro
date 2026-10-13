package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/AarishMansur/ShopKaro/backend/internal/config"
	"github.com/AarishMansur/ShopKaro/backend/internal/db"
	"github.com/AarishMansur/ShopKaro/backend/internal/store"
)

type seedProduct struct {
	Name     string
	Category string
	Price    float64
	MRP      float64
	Unit     string
	Aliases  []string
	// Weight is the average units sold per day at a day factor of 1.0.
	Weight float64
}

// Catalog mirrors a real Indian kirana shelf: fast-moving dairy and staples up
// front, slow-moving household items behind. Weights drive the demo history so
// best-seller and slow-mover charts look believable.
var catalog = []seedProduct{
	{"Amul Taaza Milk 500ml", "Dairy", 27, 27, "ltr", []string{"doodh", "milk", "amul doodh", "taaza"}, 12},
	{"Brown Bread 400g", "Fresh", 55, 55, "pkt", []string{"bread", "brown bread", "pav"}, 5},
	{"Eggs (pack of 6)", "Fresh", 54, 54, "pkt", []string{"anda", "eggs", "egg"}, 6},
	{"Maggi 2-Minute Noodles 70g", "Instant Food", 14, 14, "pkt", []string{"maggi", "noodles", "maggi noodles"}, 7},
	{"Parle-G Gold 100g", "Biscuits", 10, 10, "pkt", []string{"parle g", "parleg", "glucose", "glukoz"}, 6},
	{"Lay's Classic Salted 52g", "Snacks", 20, 20, "pkt", []string{"lays", "chips", "potato chips"}, 4.5},
	{"Kurkure Masala Munch 90g", "Snacks", 20, 20, "pkt", []string{"kurkure", "masala munch"}, 3},
	{"Bisleri Water 1L", "Beverages", 20, 20, "ltr", []string{"pani", "water", "bottle", "bisleri"}, 5},
	{"Coca-Cola 750ml", "Beverages", 40, 45, "ltr", []string{"coke", "coca cola", "cold drink", "thanda"}, 3},
	{"Tata Tea Gold 250g", "Beverages", 165, 170, "pkt", []string{"chai", "tea", "tata chai", "tata tea"}, 2.5},
	{"Onion 1kg", "Fresh", 42, 42, "kg", []string{"pyaz", "onion"}, 4},
	{"Potato 1kg", "Fresh", 32, 32, "kg", []string{"aloo", "potato"}, 4},
	{"Tomato 1kg", "Fresh", 34, 34, "kg", []string{"tamatar", "tomato"}, 3},
	{"Banana (6 pcs)", "Fresh", 40, 40, "pkt", []string{"kela", "banana"}, 3},
	{"Britannia Marie Gold 250g", "Biscuits", 35, 36, "pkt", []string{"marie", "marie gold"}, 3},
	{"Amul Dahi 400g", "Dairy", 45, 45, "pkt", []string{"dahi", "curd", "yogurt"}, 3},
	{"Haldiram's Aloo Bhujia 200g", "Snacks", 55, 58, "pkt", []string{"bhujia", "haldiram", "namkeen"}, 2.5},
	{"Coriander Leaves (bunch)", "Fresh", 10, 10, "bunch", []string{"dhaniya", "coriander", "kothmir"}, 2},
	{"Cadbury Dairy Milk 50g", "Chocolates", 45, 50, "pkt", []string{"dairy milk", "cadbury", "chocolate"}, 2},
	{"Amul Butter 100g", "Dairy", 58, 60, "pkt", []string{"butter", "amul butter"}, 2},
	{"Sunfeast Good Day 80g", "Biscuits", 15, 15, "pkt", []string{"good day", "sunfeast"}, 2},
	{"Bingo Mad Angles 66g", "Snacks", 20, 20, "pkt", []string{"bingo", "mad angles"}, 2},
	{"Apple 1kg", "Fresh", 160, 170, "kg", []string{"seb", "apple"}, 1.5},
	{"Amul Paneer 200g", "Dairy", 95, 100, "pkt", []string{"paneer", "cottage cheese"}, 1.5},
	{"Sprite 750ml", "Beverages", 40, 45, "ltr", []string{"sprite"}, 1.5},
	{"Vim Dishwash Bar 300g", "Household", 30, 32, "pcs", []string{"vim", "bartan", "dishwash"}, 1.5},
	{"Fortune Sunflower Oil 1L", "Staples", 155, 160, "ltr", []string{"tel", "oil", "sunflower oil", "fortune"}, 1.5},
	{"Tata Salt 1kg", "Spices", 28, 30, "kg", []string{"namak", "salt", "tata namak"}, 1.5},
	{"5 Star 40g", "Chocolates", 20, 20, "pkt", []string{"5 star", "five star"}, 1.5},
	{"Aashirvaad Atta 5kg", "Staples", 265, 275, "pkt", []string{"atta", "flour", "wheat flour", "aashirvaad"}, 1},
	{"India Gate Basmati Rice 1kg", "Staples", 130, 135, "kg", []string{"chawal", "rice", "basmati"}, 1},
	{"Surf Excel Easy Wash 1kg", "Household", 135, 140, "pkt", []string{"surf", "washing powder", "surf excel"}, 1},
	{"Colgate Strong Teeth 100g", "Personal Care", 58, 60, "pcs", []string{"colgate", "manjan", "toothpaste"}, 1.2},
	{"Dettol Original Soap 125g", "Personal Care", 42, 45, "pcs", []string{"dettol", "saban", "soap"}, 1},
	{"Real Mixed Fruit Juice 1L", "Beverages", 120, 125, "ltr", []string{"real", "juice", "fruit juice"}, 1},
	{"Kissan Mixed Fruit Jam 300g", "Spreads", 135, 140, "jar", []string{"jam", "kissan", "fruit jam"}, 0.8},
	{"Everest Garam Masala 50g", "Spices", 65, 68, "pkt", []string{"garam masala", "everest"}, 0.8},
	{"Dove Beauty Bar 100g", "Personal Care", 65, 70, "pcs", []string{"dove", "dove saban"}, 0.6},
	{"Everest Lal Mirch Powder 100g", "Spices", 78, 80, "pkt", []string{"lal mirch", "red chilli", "chilli powder"}, 0.6},
	{"Bru Instant Coffee 50g", "Beverages", 190, 195, "jar", []string{"bru", "coffee"}, 0.5},
	{"Dettol Handwash 200ml", "Personal Care", 99, 105, "pcs", []string{"handwash", "dettol handwash"}, 0.5},
	{"Harpic Power Plus 500ml", "Household", 100, 105, "pcs", []string{"harpic", "toilet cleaner"}, 0.4},
	{"Lizol Floor Cleaner 500ml", "Household", 115, 120, "pcs", []string{"lizol", "floor cleaner"}, 0.4},
	{"MDH Chana Masala 100g", "Spices", 92, 95, "pkt", []string{"chana masala", "mdh"}, 0.4},
}

// dayFactor mimics the kirana week: steady weekdays, a Friday bump and a
// busy weekend.
var dayFactor = map[time.Weekday]float64{
	time.Monday: 1.0, time.Tuesday: 0.95, time.Wednesday: 1.0, time.Thursday: 1.05,
	time.Friday: 1.2, time.Saturday: 1.4, time.Sunday: 1.3,
}

// hourWeight reflects the two busy bands of an Indian locality: morning
// (milk/bread run) and evening (snack run).
var hourWeight = [24]float64{0, 0, 0, 0, 0, 0.5, 2, 5, 8, 7, 5, 3.5, 3, 2, 1.5, 2, 3.5, 6, 9, 8, 6, 4, 2, 0.5}

func main() {
	reset := flag.Bool("reset", false, "wipe products and sales before seeding")
	weeks := flag.Int("weeks", 0, "weeks of history (default from SEED_WEEKS)")
	flag.Parse()

	cfg := config.Load()
	if *weeks > 0 {
		cfg.SeedWeeks = *weeks
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	var existing int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&existing); err != nil {
		log.Fatalf("check products: %v", err)
	}
	if *reset {
		if _, err := pool.Exec(ctx, `TRUNCATE sales RESTART IDENTITY; DELETE FROM products RESTART IDENTITY`); err != nil {
			log.Fatalf("reset: %v", err)
		}
		existing = 0
		log.Printf("seed: reset existing data")
	}
	if existing > 0 {
		var sales int
		_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM sales`).Scan(&sales)
		if sales > 0 {
			log.Printf("seed: %d products and %d sales already present, skipping (use -reset to reseed)", existing, sales)
			return
		}
	}

	st := store.New(pool)
	ids := make([]int64, 0, len(catalog))
	for i, p := range catalog {
		created, err := st.CreateProduct(ctx, store.ProductInput{
			Name: p.Name, Category: p.Category, Price: p.Price, MRP: p.MRP,
			Unit: p.Unit, Aliases: p.Aliases, SortOrder: i,
		})
		if err != nil {
			log.Fatalf("create product %s: %v", p.Name, err)
		}
		ids = append(ids, created.ID)
	}
	log.Printf("seed: inserted %d products", len(ids))

	rng := rand.New(rand.NewSource(20261012))
	now := time.Now().In(cfg.ShopTZ)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cfg.ShopTZ)
	startDay := today.AddDate(0, 0, -(cfg.SeedWeeks*7 - 1))

	total := 0
	seq := 0
	pending := make([]store.SaleInput, 0, 500)
	flush := func() {
		if len(pending) == 0 {
			return
		}
		if _, err := st.InsertSales(ctx, pending); err != nil {
			log.Fatalf("insert sales: %v", err)
		}
		total += len(pending)
		pending = pending[:0]
	}
	for day := startDay; !day.After(today); day = day.AddDate(0, 0, 1) {
		factor := dayFactor[day.Weekday()]
		// Gentle week-on-week growth so the revenue chart has a story.
		weekIndex := int(day.Sub(startDay).Hours() / 24 / 7)
		growth := 1.0 + float64(weekIndex)*0.015
		for i, p := range catalog {
			lambda := p.Weight * factor * growth * (0.75 + 0.5*rng.Float64())
			qty := poisson(rng, lambda)
			if qty == 0 {
				continue
			}
			for _, chunk := range splitQty(rng, qty) {
				hour := sampleHour(rng)
				minute := rng.Intn(60)
				soldAt := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, cfg.ShopTZ)
				if soldAt.After(now) {
					continue
				}
				seq++
				price := p.Price
				if rng.Float64() < 0.15 {
					price = math.Round((p.Price*0.95)*10) / 10 // occasional discount
				}
				sold := soldAt
				pending = append(pending, store.SaleInput{
					ClientID:  fmt.Sprintf("seed-%05d", seq),
					ProductID: ids[i],
					Qty:       chunk,
					UnitPrice: &price,
					SoldAt:    &sold,
					Source:    "seed",
				})
				if len(pending) >= 500 {
					flush()
				}
			}
		}
	}
	flush()
	log.Printf("seed: inserted %d sales across %d days (%d weeks)", total, int(today.Sub(startDay).Hours()/24)+1, cfg.SeedWeeks)
}

func poisson(rng *rand.Rand, lambda float64) int {
	if lambda <= 0 {
		return 0
	}
	l := math.Exp(-lambda)
	k, p := 0, 1.0
	for {
		k++
		p *= rng.Float64()
		if p <= l {
			return k - 1
		}
	}
}

// splitQty breaks a day's units into 1-3 receipts the way customers actually
// buy (one person takes 2, another takes 1).
func splitQty(rng *rand.Rand, qty int) []int {
	if qty <= 2 {
		return []int{qty}
	}
	parts := 1 + rng.Intn(3)
	if parts > qty {
		parts = qty
	}
	out := make([]int, parts)
	for i := range out {
		out[i] = 1
	}
	remaining := qty - parts
	for remaining > 0 {
		out[rng.Intn(parts)]++
		remaining--
	}
	return out
}

func sampleHour(rng *rand.Rand) int {
	var total float64
	for _, w := range hourWeight {
		total += w
	}
	pick := rng.Float64() * total
	for h, w := range hourWeight {
		pick -= w
		if pick <= 0 {
			return h
		}
	}
	return 19
}
