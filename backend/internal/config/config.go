package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port     string
	ShopName string
	ShopTZ   *time.Location

	DatabaseURL string
	CORSOrigins string

	// Gemma via Ollama (open-weight). Empty URL disables the AI features.
	OllamaURL        string
	GemmaModel       string
	GemmaVisionModel string

	// Open-source Whisper sidecar for voice entry.
	WhisperURL string

	// TabPFN (Prior Labs) for the reorder forecast.
	TabPFNBaseURL string
	TabPFNAPIKey  string

	// Temporal durable workflows for the scheduled jobs.
	TemporalEnabled   bool
	TemporalAddress   string
	TemporalNamespace string

	SeedWeeks int
	// SummaryLang is the default language for the scheduled weekly recap.
	SummaryLang string
}

func Load() *Config {
	cfg := &Config{
		Port:              env("PORT", "8080"),
		ShopName:          env("SHOP_NAME", "Sharma Kirana Store"),
		DatabaseURL:       env("DATABASE_URL", "postgres://shopkaro:shopkaro@localhost:5432/shopkaro?sslmode=disable"),
		CORSOrigins:       env("CORS_ORIGINS", "*"),
		OllamaURL:         env("OLLAMA_URL", ""),
		GemmaModel:        env("GEMMA_MODEL", "gemma3:4b"),
		GemmaVisionModel:  env("GEMMA_VISION_MODEL", ""),
		WhisperURL:        env("WHISPER_URL", ""),
		TabPFNBaseURL:     env("TABPFN_BASE_URL", ""),
		TabPFNAPIKey:      env("TABPFN_API_KEY", ""),
		TemporalEnabled:   envBool("TEMPORAL_ENABLED", false),
		TemporalAddress:   env("TEMPORAL_ADDRESS", "localhost:7233"),
		TemporalNamespace: env("TEMPORAL_NAMESPACE", "default"),
		SeedWeeks:         envInt("SEED_WEEKS", 10),
		SummaryLang:       env("SUMMARY_LANG", "en"),
	}
	if cfg.GemmaVisionModel == "" {
		cfg.GemmaVisionModel = cfg.GemmaModel
	}
	tz, err := time.LoadLocation(env("SHOP_TZ", "Asia/Kolkata"))
	if err != nil {
		tz = time.FixedZone("IST", 5*60*60+30*60)
	}
	cfg.ShopTZ = tz
	return cfg
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
