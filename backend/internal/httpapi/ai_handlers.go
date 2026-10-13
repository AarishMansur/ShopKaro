package httpapi

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"

	"github.com/AarishMansur/ShopKaro/backend/internal/ai"
)

func (s *Server) handleVoice(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "expected multipart audio")
		return
	}
	file, header, err := r.FormFile("audio")
	if err != nil {
		writeError(w, http.StatusBadRequest, "audio field is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 15<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read audio")
		return
	}

	name := header.Filename
	if name == "" {
		name = "voice.webm"
	}
	text, err := ai.Transcribe(r.Context(), s.whisper, name, data)
	if err != nil {
		if errors.Is(err, ai.ErrWhisperDisabled) {
			writeJSON(w, http.StatusNotImplemented, map[string]string{
				"error": "whisper is not configured",
				"code":  "whisper_disabled",
			})
			return
		}
		writeError(w, http.StatusBadGateway, "transcription failed: "+err.Error())
		return
	}

	products, err := s.st.ListProducts(r.Context(), false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	res, err := ai.Parse(r.Context(), s.ollama, products, text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	res.Transcript = text
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	lang := queryStr(r, "lang", s.cfg.SummaryLang)
	stats, err := ai.BuildWeeklyStats(r.Context(), s.st, s.cfg.ShopTZ.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	content, engine, err := ai.WeeklySummary(r.Context(), s.ollama, stats, lang)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lang":    lang,
		"engine":  engine,
		"content": content,
		"stats":   stats,
	})
}

func (s *Server) handleProductFromPhoto(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 15<<20)
	if err := r.ParseMultipartForm(15 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "expected multipart photo")
		return
	}
	file, _, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "photo field is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read photo")
		return
	}
	suggestion, err := ai.ProductFromPhoto(r.Context(), s.ollama, base64.StdEncoding.EncodeToString(data))
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Gemma vision is unavailable — add the product manually",
			"code":  "gemma_unavailable",
		})
		return
	}
	writeJSON(w, http.StatusOK, suggestion)
}
