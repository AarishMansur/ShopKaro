package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/AarishMansur/ShopKaro/backend/internal/jobs"
	"github.com/AarishMansur/ShopKaro/backend/internal/models"
	"github.com/AarishMansur/ShopKaro/backend/internal/store"
)

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	includeInactive := queryStr(r, "include_inactive", "") == "1"
	products, err := s.st.ListProducts(r.Context(), includeInactive)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var in store.ProductInput
	if err := decodeLoose(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p, err := s.st.CreateProduct(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	var in store.ProductInput
	if err := decodeLoose(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	p, err := s.st.UpdateProduct(r.Context(), id, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	result, err := s.st.DeleteProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": result})
}

func (s *Server) handleListSales(w http.ResponseWriter, r *http.Request) {
	sales, err := s.st.ListSales(r.Context(), queryInt(r, "limit", 30))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sales": sales})
}

func (s *Server) handleCreateSales(w http.ResponseWriter, r *http.Request) {
	// Sniff whether this is a bare sale object, a single wrapper or a batch.
	var probe map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&probe); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	inputs := []store.SaleInput{}
	if raw, ok := probe["sales"]; ok {
		if err := json.Unmarshal(raw, &inputs); err != nil {
			writeError(w, http.StatusBadRequest, "invalid sales array")
			return
		}
	} else if raw, ok := probe["sale"]; ok {
		var one store.SaleInput
		if err := json.Unmarshal(raw, &one); err != nil {
			writeError(w, http.StatusBadRequest, "invalid sale")
			return
		}
		inputs = []store.SaleInput{one}
	} else {
		var one store.SaleInput
		raw, _ := json.Marshal(probe)
		if err := json.Unmarshal(raw, &one); err != nil || one.ClientID == "" {
			writeError(w, http.StatusBadRequest, "expected a sale object")
			return
		}
		inputs = []store.SaleInput{one}
	}

	sales, err := s.st.InsertSales(r.Context(), inputs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sales": sales, "accepted": len(sales)})
}

func (s *Server) handleUpdateSale(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	var req struct {
		Qty int `json:"qty"`
	}
	if err := decodeLoose(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	sale, err := s.st.UpdateSaleQty(r.Context(), id, req.Qty)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sale)
}

func (s *Server) handleDeleteSale(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.st.DeleteSale(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "sale not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (s *Server) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	days := queryInt(r, "days", 7)
	sum, err := s.st.Summary(r.Context(), days, s.cfg.ShopTZ.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) handleBestSellers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.st.BestSellers(r.Context(), queryInt(r, "days", 30), queryInt(r, "limit", 10))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": rows})
}

func (s *Server) handleSlowMovers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.st.SlowMovers(r.Context(), queryInt(r, "days", 30), queryInt(r, "limit", 10))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": rows})
}

func (s *Server) handleByDay(w http.ResponseWriter, r *http.Request) {
	rows, err := s.st.ByDay(r.Context(), queryInt(r, "days", 28), s.cfg.ShopTZ.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": rows})
}

func (s *Server) handleByHour(w http.ResponseWriter, r *http.Request) {
	rows, err := s.st.ByHour(r.Context(), queryInt(r, "days", 28), s.cfg.ShopTZ.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hours": rows})
}

func (s *Server) handleForecast(w http.ResponseWriter, r *http.Request) {
	fc, err := s.fc.Forecast(r.Context(), queryInt(r, "weeks_ahead", 1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, fc)
}

func (s *Server) handleJobsStatus(w http.ResponseWriter, r *http.Request) {
	status := models.JobStatus{
		Mode:     s.mode,
		Enabled:  true,
		LastRuns: map[string]models.JobRun{},
		NextRuns: map[string]string{},
	}
	if s.temporal {
		status.Detail = "Temporal schedules run the weekly summary and reorder report durably."
	} else {
		status.Detail = "Built-in scheduler (enable TEMPORAL_ENABLED=true for durable Temporal workflows)."
	}
	for _, name := range []string{jobs.WeeklySummaryJob, jobs.ReorderForecastJob} {
		run, err := s.st.JobRun(r.Context(), name)
		if err == nil {
			status.LastRuns[name] = run
		}
		next := jobs.NextDue(name, time.Now(), s.cfg.ShopTZ)
		status.NextRuns[name] = next.Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleRunJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind string `json:"kind"`
		Lang string `json:"lang"`
	}
	if err := decodeLoose(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Lang == "" {
		req.Lang = s.cfg.SummaryLang
	}
	var report *models.Report
	var err error
	switch req.Kind {
	case jobs.WeeklySummaryJob, "summary":
		report, err = jobs.RunWeeklySummary(r.Context(), s.deps, req.Lang)
		jobs.Mark(r.Context(), s.st, jobs.WeeklySummaryJob, err)
	case jobs.ReorderForecastJob, "forecast":
		report, err = jobs.RunReorderForecast(r.Context(), s.deps)
		jobs.Mark(r.Context(), s.st, jobs.ReorderForecastJob, err)
	default:
		writeError(w, http.StatusBadRequest, "kind must be weekly_summary or reorder_forecast")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleLatestReport(w http.ResponseWriter, r *http.Request) {
	kind := queryStr(r, "kind", jobs.WeeklySummaryJob)
	report, err := s.st.LatestReport(r.Context(), kind)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"report": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"report": report})
}
