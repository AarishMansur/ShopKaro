package ai

import (
	"testing"

	"github.com/AarishMansur/ShopKaro/backend/internal/models"
)

func fixtures() []models.Product {
	return []models.Product{
		{ID: 1, Name: "Maggi 2-Minute Noodles 70g", Price: 14, Aliases: []string{"maggi", "noodles"}},
		{ID: 2, Name: "Amul Taaza Milk 500ml", Price: 27, Aliases: []string{"doodh", "milk", "taaza"}},
		{ID: 3, Name: "Parle-G Gold 100g", Price: 10, Aliases: []string{"parle g", "parleg", "glucose"}},
		{ID: 4, Name: "Tata Salt 1kg", Price: 28, Aliases: []string{"namak", "salt"}},
	}
}

func TestParseRuleEngine(t *testing.T) {
	cases := []struct {
		in      string
		wantIDs map[int64]int
	}{
		{"2 Maggi, 1 doodh", map[int64]int{1: 2, 2: 1}},
		{"do parle g", map[int64]int{3: 2}},
		{"ek namak, do doodh", map[int64]int{4: 1, 2: 2}},
		{"maggi", map[int64]int{1: 1}},
		{"3x maggi aur 2 milk", map[int64]int{1: 3, 2: 2}},
		{"maggie", map[int64]int{1: 1}},
	}
	for _, tc := range cases {
		res := parseWithRules(fixtures(), tc.in)
		got := map[int64]int{}
		for _, e := range res.Entries {
			got[e.ProductID] = e.Qty
		}
		if len(got) != len(tc.wantIDs) {
			t.Errorf("%q: got %d entries, want %d (%+v)", tc.in, len(got), len(tc.wantIDs), got)
			continue
		}
		for id, qty := range tc.wantIDs {
			if got[id] != qty {
				t.Errorf("%q: product %d qty = %d, want %d", tc.in, id, got[id], qty)
			}
		}
	}
}

func TestParseKeepsUnmatchedSegments(t *testing.T) {
	res := parseWithRules(fixtures(), "2 maggi, kuch aur")
	if len(res.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %+v", res.Entries)
	}
	if len(res.Unmatched) != 1 || res.Unmatched[0] != "kuch aur" {
		t.Fatalf("expected unmatched segment, got %+v", res.Unmatched)
	}
}

func TestLevenshtein(t *testing.T) {
	if d := levenshtein("maggi", "maggie"); d != 1 {
		t.Errorf("distance = %d, want 1", d)
	}
	if d := levenshtein("salt", "salt"); d != 0 {
		t.Errorf("distance = %d, want 0", d)
	}
}
