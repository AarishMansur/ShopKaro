package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/AarishMansur/ShopKaro/backend/internal/ai"
	"github.com/AarishMansur/ShopKaro/backend/internal/config"
	"github.com/AarishMansur/ShopKaro/backend/internal/forecast"
	"github.com/AarishMansur/ShopKaro/backend/internal/jobs"
	"github.com/AarishMansur/ShopKaro/backend/internal/store"
)

type Server struct {
	cfg      *config.Config
	st       *store.Store
	ollama   *ai.Ollama
	fc       *forecast.Service
	deps     *jobs.Deps
	whisper  string
	temporal bool
	mode     string
}

func New(cfg *config.Config, st *store.Store, ollama *ai.Ollama, fc *forecast.Service, deps *jobs.Deps) *Server {
	return &Server{cfg: cfg, st: st, ollama: ollama, fc: fc, deps: deps, whisper: cfg.WhisperURL, mode: "builtin"}
}

func (s *Server) SetTemporal(active bool, detail string) {
	s.temporal = active
	s.mode = "builtin"
	if active {
		s.mode = "temporal"
	}
	if detail != "" {
		s.mode = detail
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/features", s.handleFeatures)

	mux.HandleFunc("GET /api/products", s.handleListProducts)
	mux.HandleFunc("POST /api/products", s.handleCreateProduct)
	mux.HandleFunc("PUT /api/products/{id}", s.handleUpdateProduct)
	mux.HandleFunc("DELETE /api/products/{id}", s.handleDeleteProduct)
	mux.HandleFunc("POST /api/products/from-photo", s.handleProductFromPhoto)

	mux.HandleFunc("GET /api/sales", s.handleListSales)
	mux.HandleFunc("POST /api/sales", s.handleCreateSales)
	mux.HandleFunc("POST /api/sales/batch", s.handleCreateSales)
	mux.HandleFunc("PATCH /api/sales/{id}", s.handleUpdateSale)
	mux.HandleFunc("DELETE /api/sales/{id}", s.handleDeleteSale)

	mux.HandleFunc("GET /api/dashboard/summary", s.handleDashboardSummary)
	mux.HandleFunc("GET /api/dashboard/best-sellers", s.handleBestSellers)
	mux.HandleFunc("GET /api/dashboard/slow-movers", s.handleSlowMovers)
	mux.HandleFunc("GET /api/dashboard/by-day", s.handleByDay)
	mux.HandleFunc("GET /api/dashboard/by-hour", s.handleByHour)

	mux.HandleFunc("GET /api/reorder/forecast", s.handleForecast)

	mux.HandleFunc("POST /api/ai/parse", s.handleParse)
	mux.HandleFunc("POST /api/ai/voice", s.handleVoice)
	mux.HandleFunc("GET /api/ai/summary", s.handleSummary)

	mux.HandleFunc("GET /api/jobs/status", s.handleJobsStatus)
	mux.HandleFunc("POST /api/jobs/run", s.handleRunJob)
	mux.HandleFunc("GET /api/reports/latest", s.handleLatestReport)

	return withRecovery(withCORS(s.cfg.CORSOrigins)(mux))
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v\n%s", rec, debug.Stack())
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func withCORS(origins string) func(http.Handler) http.Handler {
	allowed := "*"
	if origins != "" {
		allowed = origins
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

// decodeLoose keeps forward compatibility for clients sending extra fields.
func decodeLoose(r *http.Request, dst any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(dst)
}

func queryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	var n int
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func queryStr(r *http.Request, key, fallback string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return fallback
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().UTC()})
}

func (s *Server) handleFeatures(w http.ResponseWriter, r *http.Request) {
	var timescale bool
	_ = s.st.Pool().QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')`).Scan(&timescale)

	gemma := s.ollama.Available(r.Context())
	var gemmaErr string
	if s.ollama.Configured() && !gemma {
		gemmaErr = "OLLAMA_URL is set but the server did not answer"
	}
	whisper := s.whisper != ""
	writeJSON(w, http.StatusOK, map[string]any{
		"gemma":        gemma,
		"gemma_config": s.ollama.Configured(),
		"gemma_model":  s.ollama.Model(),
		"gemma_error":  gemmaErr,
		"whisper":      whisper,
		"tabpfn":       s.fc != nil && s.fc.Method() == "tabpfn",
		"temporal":     s.temporal,
		"scheduler":    s.mode,
		"timescaledb":  timescale,
		"shop":         map[string]any{"name": s.cfg.ShopName, "tz": s.cfg.ShopTZ.String()},
	})
}

func (s *Server) handleParse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := decodeLoose(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	products, err := s.st.ListProducts(r.Context(), false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	res, err := ai.Parse(r.Context(), s.ollama, products, req.Text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func isWhisperDisabled(err error) bool {
	return errors.Is(err, ai.ErrWhisperDisabled) || strings.Contains(err.Error(), "whisper not configured")
}
