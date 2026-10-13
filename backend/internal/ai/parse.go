package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/AarishMansur/ShopKaro/backend/internal/models"
)

type ParsedEntry struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Qty       int     `json:"qty"`
	UnitPrice float64 `json:"unit_price"`
	MatchedOn string  `json:"matched_on,omitempty"`
}

type ParseResult struct {
	Engine     string         `json:"engine"` // "gemma" or "rules"
	Entries    []ParsedEntry  `json:"entries"`
	Unmatched  []string       `json:"unmatched"`
	Transcript string         `json:"transcript,omitempty"`
}

var numberWords = map[string]int{
	"ek": 1, "one": 1, "do": 2, "two": 2, "teen": 3, "tin": 3, "three": 3,
	"char": 4, "chaar": 4, "four": 4, "panch": 5, "paanch": 5, "five": 5,
	"chhe": 6, "che": 6, "six": 6, "saat": 7, "seven": 7, "aath": 8, "eight": 8,
	"nau": 9, "nine": 9, "dus": 10, "ten": 10, "pandra": 15, "bees": 20, "twenty": 20,
	"pachas": 50, "sau": 100, "hundred": 100,
}

var segmentSplit = regexp.MustCompile(`[,;|]|\s+(?:aur|or|and)\s+|।`)

// Parse converts a Hinglish/English phrase like "2 Maggi, 1 doodh" into sale
// entries. Gemma is used when available; the rule engine always works offline.
func Parse(ctx context.Context, o *Ollama, products []models.Product, text string) (*ParseResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return &ParseResult{Engine: "rules"}, nil
	}
	if o.Available(ctx) {
		res, err := parseWithGemma(ctx, o, products, text)
		if err == nil && len(res.Entries) > 0 {
			res.Transcript = text
			return res, nil
		}
	}
	res := parseWithRules(products, text)
	res.Transcript = text
	return res, nil
}

func parseWithGemma(ctx context.Context, o *Ollama, products []models.Product, text string) (*ParseResult, error) {
	var catalog strings.Builder
	for _, p := range products {
		fmt.Fprintf(&catalog, "%d|%s|%s|%s|%.2f\n", p.ID, p.Name, p.Category, strings.Join(p.Aliases, "/"), p.Price)
	}
	system := `You convert a shopkeeper's spoken or typed shopping phrase into product entries.
You receive a catalog with lines: id|name|category|aliases|price.
Return ONLY JSON: {"entries":[{"product_id":<id>,"qty":<int>}],"unmatched":["<leftover text>"]}.
Rules: quantities default to 1; match Hinglish words (doodh->milk, namak->salt, saban->soap, pani->water, aloo->potato, pyaz->onion, kela->banana, seb->apple, anda->egg, chawal->rice, atta->flour, chai->tea, tel->oil, manjan->toothpaste, dhaniya->coriander, glukoz->glucose). Never invent ids.`

	user := "CATALOG:\n" + catalog.String() + "\nPHRASE:\n" + text
	out, err := o.ChatJSON(ctx, o.Model(), system, user)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Entries []struct {
			ProductID int64 `json:"product_id"`
			Qty       int   `json:"qty"`
		} `json:"entries"`
		Unmatched []string `json:"unmatched"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, fmt.Errorf("gemma returned invalid json: %w", err)
	}
	byID := map[int64]models.Product{}
	for _, p := range products {
		byID[p.ID] = p
	}
	res := &ParseResult{Engine: "gemma", Unmatched: []string{}}
	for _, e := range parsed.Entries {
		p, ok := byID[e.ProductID]
		if !ok {
			continue
		}
		qty := e.Qty
		if qty <= 0 {
			qty = 1
		}
		res.Entries = append(res.Entries, ParsedEntry{
			ProductID: p.ID, Name: p.Name, Qty: qty, UnitPrice: p.Price, MatchedOn: "gemma",
		})
	}
	res.Unmatched = parsed.Unmatched
	if len(res.Entries) == 0 {
		return nil, fmt.Errorf("gemma matched no products")
	}
	return res, nil
}

func parseWithRules(products []models.Product, text string) *ParseResult {
	res := &ParseResult{Engine: "rules", Unmatched: []string{}}
	lower := strings.ToLower(text)
	segments := segmentSplit.Split(lower, -1)
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		qty, cleaned := extractQty(seg)
		product, matched := matchProduct(products, cleaned)
		if product == nil {
			res.Unmatched = append(res.Unmatched, strings.TrimSpace(seg))
			continue
		}
		res.Entries = append(res.Entries, ParsedEntry{
			ProductID: product.ID, Name: product.Name, Qty: qty, UnitPrice: product.Price, MatchedOn: matched,
		})
	}
	return res
}

func extractQty(seg string) (int, string) {
	if m := regexp.MustCompile(`\b(\d+)\s*(?:x|nos|pcs|packets?|pkts?|pieces?)?\b`).FindStringSubmatch(seg); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 && n < 1000 {
			rest := strings.Replace(seg, m[0], " ", 1)
			return n, rest
		}
	}
	tokens := strings.Fields(seg)
	for i, t := range tokens {
		if n, ok := numberWords[t]; ok {
			rest := strings.Join(append(append([]string{}, tokens[:i]...), tokens[i+1:]...), " ")
			return n, rest
		}
	}
	return 1, seg
}

// matchProduct scores every product against the phrase fragment and returns the
// best alias hit (longest match wins, fuzzy single-token typos allowed).
func matchProduct(products []models.Product, seg string) (*models.Product, string) {
	seg = " " + strings.Join(strings.Fields(seg), " ") + " "
	type scored struct {
		p     *models.Product
		match string
		score int
	}
	best := scored{}
	for i := range products {
		p := &products[i]
		candidates := append([]string{p.Name}, p.Aliases...)
		for _, c := range candidates {
			c = strings.ToLower(strings.TrimSpace(c))
			if c == "" {
				continue
			}
			if strings.Contains(seg, " "+c+" ") || strings.Contains(seg, " "+c) || strings.Contains(seg, c+" ") {
				if len(c) > best.score {
					best = scored{p: p, match: c, score: len(c)}
				}
			}
		}
	}
	if best.p != nil {
		return best.p, best.match
	}
	// Fuzzy pass for spoken typos: "maggie" -> "maggi".
	segTokens := strings.Fields(seg)
	for i := range products {
		p := &products[i]
		names := append([]string{p.Name}, p.Aliases...)
		for _, name := range names {
			for _, nt := range strings.Fields(strings.ToLower(name)) {
				if len(nt) < 3 {
					continue
				}
				for _, st := range segTokens {
					if levenshtein(nt, st) <= 1 && abs(len(nt)-len(st)) <= 1 {
						return p, st
					}
				}
			}
		}
	}
	return nil, ""
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func min(nums ...int) int {
	m := nums[0]
	for _, n := range nums[1:] {
		if n < m {
			m = n
		}
	}
	return m
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// SortEntries keeps output stable for the confirm screen.
func SortEntries(entries []ParsedEntry) {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
}
