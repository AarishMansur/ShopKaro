package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Ollama talks to a local Ollama server serving an open-weight Gemma model.
// Every feature is optional: with no OLLAMA_URL configured, callers fall back
// to the rule-based implementations in this package.
type Ollama struct {
	url         string
	model       string
	visionModel string
	client      *http.Client
}

func NewOllama(url, model, visionModel string) *Ollama {
	return &Ollama{
		url:         strings.TrimRight(url, "/"),
		model:       model,
		visionModel: visionModel,
		client:      &http.Client{Timeout: 60 * time.Second},
	}
}

func (o *Ollama) Configured() bool { return o != nil && o.url != "" }

func (o *Ollama) Model() string       { return o.model }
func (o *Ollama) VisionModel() string { return o.visionModel }

// Available pings the server so feature flags reflect reality.
func (o *Ollama) Available(ctx context.Context) bool {
	if !o.Configured() {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.url+"/api/tags", nil)
	if err != nil {
		return false
	}
	res, err := o.client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK
}

type chatMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	Stream    bool          `json:"stream"`
	Format    string        `json:"format,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
}

type chatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Error string `json:"error"`
}

func (o *Ollama) chat(ctx context.Context, model, system, user string, images []string, jsonMode bool) (string, error) {
	if !o.Configured() {
		return "", fmt.Errorf("ollama not configured")
	}
	body := chatRequest{
		Model:  model,
		Stream: false,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user, Images: images},
		},
		Options: map[string]any{"temperature": 0.1},
	}
	if jsonMode {
		body.Format = "json"
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var out chatResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		if out.Error != "" {
			return "", fmt.Errorf("ollama: %s", out.Error)
		}
		return "", fmt.Errorf("ollama: status %d", res.StatusCode)
	}
	return strings.TrimSpace(out.Message.Content), nil
}

// ChatJSON asks the model for a JSON document.
func (o *Ollama) ChatJSON(ctx context.Context, model, system, user string) (string, error) {
	return o.chat(ctx, model, system, user, nil, true)
}

// ChatVision asks a vision-capable Gemma model about an image (base64, no prefix).
func (o *Ollama) ChatVision(ctx context.Context, model, system, user, imageB64 string) (string, error) {
	return o.chat(ctx, model, system, user, []string{imageB64}, true)
}

// ChatText is a plain conversation turn (used for the weekly summary).
func (o *Ollama) ChatText(ctx context.Context, model, system, user string) (string, error) {
	return o.chat(ctx, model, system, user, nil, false)
}
