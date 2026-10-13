package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ProductSuggestion struct {
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Price    float64  `json:"price"`
	MRP      float64  `json:"mrp"`
	Unit     string   `json:"unit"`
	Aliases  []string `json:"aliases"`
	Emoji    string   `json:"emoji"`
}

// ProductFromPhoto asks a vision-capable Gemma model to read a pack photo and
// prefill the add-product form.
func ProductFromPhoto(ctx context.Context, o *Ollama, imageB64 string) (*ProductSuggestion, error) {
	if !o.Available(ctx) {
		return nil, fmt.Errorf("gemma unavailable")
	}
	system := `You read a photo of a consumer product pack (usually Indian grocery).
Return ONLY JSON: {"name":"short shelf name","category":"one of Biscuits,Snacks,Chocolates,Dairy,Beverages,Staples,Spices,Fresh,Personal Care,Household,Instant Food,Other","price":number,"mrp":number,"unit":"pcs|ml|g|kg|ltr|pkt","aliases":["hinglish words a shopkeeper might say"]}.
Use ₹ prices typical for India. If the price is not visible, estimate a common MRP. Keep name under 24 chars.`
	user := "What product is this? Return the JSON."
	out, err := o.ChatVision(ctx, o.VisionModel(), system, user, imageB64)
	if err != nil {
		return nil, err
	}
	var s ProductSuggestion
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		return nil, fmt.Errorf("gemma returned invalid json: %w", err)
	}
	if strings.TrimSpace(s.Name) == "" {
		return nil, fmt.Errorf("gemma could not read the pack")
	}
	if s.Category == "" {
		s.Category = "Other"
	}
	if s.Unit == "" {
		s.Unit = "pcs"
	}
	if s.Aliases == nil {
		s.Aliases = []string{}
	}
	return &s, nil
}
