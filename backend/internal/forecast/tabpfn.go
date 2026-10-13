package forecast

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TabPFN calls the Prior Labs REST API (TabPFN-3.5) for the weekly regression.
// Any failure falls back to the statistical forecast, so the app never blocks.
type TabPFN struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewTabPFN(baseURL, apiKey string) *TabPFN {
	return &TabPFN{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 4 * time.Minute},
	}
}

func (t *TabPFN) Enabled() bool {
	return t != nil && t.baseURL != "" && t.apiKey != ""
}

var featureHeader = []string{"last_week", "avg_4w", "avg_8w", "std", "slope", "weekend_ratio", "active_ratio", "price", "category"}

// featuresFor builds the tabular row for a product using only the completed
// weeks up to cutoff (exclusive), mirroring a supervised learning problem.
func featuresFor(sr series, cutoff int) []string {
	q := sr.qty
	if cutoff > len(q) {
		cutoff = len(q)
	}
	look := q[:cutoff]
	last := 0.0
	_, avg4, avg8, _ := tailStats(look)
	if len(look) > 0 {
		last = float64(look[len(look)-1])
	}
	var variance float64
	if len(look) > 0 {
		var sum float64
		for _, v := range look {
			sum += float64(v)
		}
		mean := sum / float64(len(look))
		for _, v := range look {
			variance += (float64(v) - mean) * (float64(v) - mean)
		}
		variance /= float64(len(look))
	}
	active := 0
	for _, v := range look {
		if v > 0 {
			active++
		}
	}
	activeRatio := 0.0
	if len(look) > 0 {
		activeRatio = float64(active) / float64(len(look))
	}
	return []string{
		fmt.Sprintf("%.4f", last),
		fmt.Sprintf("%.4f", avg4),
		fmt.Sprintf("%.4f", avg8),
		fmt.Sprintf("%.4f", variance),
		fmt.Sprintf("%.4f", slope(look)),
		fmt.Sprintf("%.4f", sr.weekendRatio),
		fmt.Sprintf("%.4f", activeRatio),
		fmt.Sprintf("%.4f", sr.product.Price),
		sr.product.Category,
	}
}

func buildTrainRows(list []series, nWeeks int) (xRows, yRows [][]string) {
	for cutoff := minWeeks; cutoff < nWeeks; cutoff++ {
		for _, sr := range list {
			xRows = append(xRows, featuresFor(sr, cutoff))
			yRows = append(yRows, []string{fmt.Sprintf("%d", sr.qty[cutoff])})
		}
	}
	return xRows, yRows
}

func buildTestRows(list []series, nWeeks int) [][]string {
	rows := make([][]string, 0, len(list))
	for _, sr := range list {
		rows = append(rows, featuresFor(sr, nWeeks))
	}
	return rows
}

func (t *TabPFN) Predict(ctx context.Context, list []series) ([]float64, error) {
	nWeeks := 0
	if len(list) > 0 {
		nWeeks = len(list[0].qty)
	}
	if nWeeks <= minWeeks {
		return nil, fmt.Errorf("not enough weeks for tabpfn")
	}
	xTrain, yTrain := buildTrainRows(list, nWeeks)
	if len(xTrain) == 0 {
		return nil, fmt.Errorf("no training rows")
	}
	xTest := buildTestRows(list, nWeeks)

	trainCSV := mustCSV(featureHeader, xTrain)
	targetCSV := mustCSV([]string{"qty"}, yTrain)
	testCSV := mustCSV(featureHeader, xTest)

	trainID, err := t.uploadTrainSet(ctx, trainCSV, targetCSV)
	if err != nil {
		return nil, fmt.Errorf("upload train set: %w", err)
	}
	fittedID, err := t.fit(ctx, trainID)
	if err != nil {
		return nil, fmt.Errorf("fit: %w", err)
	}
	testID, err := t.uploadTestSet(ctx, fittedID, testCSV)
	if err != nil {
		return nil, fmt.Errorf("upload test set: %w", err)
	}
	return t.predict(ctx, fittedID, testID)
}

type signedUpload struct {
	SignedURLs     []string          `json:"signed_urls"`
	RequiredHeaders map[string]string `json:"required_headers"`
}

type trainUploadResponse struct {
	TrainSetUploadID string         `json:"train_set_upload_id"`
	XTrainInfo       signedUpload   `json:"x_train_info"`
	YTrainInfo       signedUpload   `json:"y_train_info"`
}

type testUploadResponse struct {
	TestSetUploadID string       `json:"test_set_upload_id"`
	XTestInfo       signedUpload `json:"x_test_info"`
}

func (t *TabPFN) postJSON(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	res, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("%s: status %d: %s", path, res.StatusCode, truncate(string(data), 300))
	}
	return json.Unmarshal(data, out)
}

func (t *TabPFN) uploadTrainSet(ctx context.Context, xCSV, yCSV []byte) (string, error) {
	var prep trainUploadResponse
	err := t.postJSON(ctx, "/tabpfn/prepare_train_set_upload", map[string]any{
		"x_train_info": map[string]any{"format": "csv"},
		"y_train_info": map[string]any{"format": "csv"},
	}, &prep)
	if err != nil {
		return "", err
	}
	if err := t.putCSV(ctx, prep.XTrainInfo, xCSV); err != nil {
		return "", err
	}
	if err := t.putCSV(ctx, prep.YTrainInfo, yCSV); err != nil {
		return "", err
	}
	return prep.TrainSetUploadID, nil
}

func (t *TabPFN) fit(ctx context.Context, trainID string) (string, error) {
	var out struct {
		FittedTrainSetID string `json:"fitted_train_set_id"`
		Status           string `json:"status"`
	}
	err := t.postJSON(ctx, "/tabpfn/fit", map[string]any{
		"train_set_upload_id": trainID,
		"task_config": map[string]any{
			"task":           "regression",
			"tabpfn_config":  map[string]any{"model_path": "v3.5_default"},
		},
	}, &out)
	if err != nil {
		return "", err
	}
	if out.FittedTrainSetID == "" {
		return "", fmt.Errorf("fit returned no model id")
	}
	return out.FittedTrainSetID, nil
}

func (t *TabPFN) uploadTestSet(ctx context.Context, fittedID string, xCSV []byte) (string, error) {
	var prep testUploadResponse
	err := t.postJSON(ctx, "/tabpfn/prepare_test_set_upload", map[string]any{
		"fitted_train_set_id": fittedID,
		"x_test_info":         map[string]any{"format": "csv"},
	}, &prep)
	if err != nil {
		return "", err
	}
	if err := t.putCSV(ctx, prep.XTestInfo, xCSV); err != nil {
		return "", err
	}
	return prep.TestSetUploadID, nil
}

func (t *TabPFN) predict(ctx context.Context, fittedID, testID string) ([]float64, error) {
	var out struct {
		Prediction json.RawMessage `json:"prediction"`
	}
	err := t.postJSON(ctx, "/tabpfn/predict", map[string]any{
		"test_set_upload_id":  testID,
		"fitted_train_set_id": fittedID,
		"task_config": map[string]any{
			"task":            "regression",
			"predict_params":  map[string]any{"output_type": "mean"},
		},
	}, &out)
	if err != nil {
		return nil, err
	}
	return parsePredictions(out.Prediction)
}

func parsePredictions(raw json.RawMessage) ([]float64, error) {
	var flat []float64
	if err := json.Unmarshal(raw, &flat); err == nil {
		return flat, nil
	}
	var nested [][]float64
	if err := json.Unmarshal(raw, &nested); err == nil {
		out := make([]float64, 0, len(nested))
		for _, row := range nested {
			if len(row) > 0 {
				out = append(out, row[0])
			} else {
				out = append(out, 0)
			}
		}
		return out, nil
	}
	return nil, fmt.Errorf("unexpected prediction shape")
}

func (t *TabPFN) putCSV(ctx context.Context, info signedUpload, data []byte) error {
	if len(info.SignedURLs) == 0 {
		return fmt.Errorf("no signed url returned")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, info.SignedURLs[0], bytes.NewReader(data))
	if err != nil {
		return err
	}
	for k, v := range info.RequiredHeaders {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "text/csv")
	}
	res, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("csv upload: status %d: %s", res.StatusCode, truncate(string(body), 200))
	}
	return nil
}

func mustCSV(header []string, rows [][]string) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(header)
	_ = w.WriteAll(rows)
	w.Flush()
	return buf.Bytes()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
