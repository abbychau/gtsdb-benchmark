package main

import (
	"bytes"
	"context"
	"encoding/json"

	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/VictoriaMetrics/metrics"
)

type vmDriver struct {
	url    string
	client *http.Client
}

func newVMDriver(url string) *vmDriver {
	return &vmDriver{
		url: strings.TrimRight(url, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 100,
				MaxConnsPerHost:     100,
			},
		},
	}
}

func (d *vmDriver) Name() string { return "VM" }

func (d *vmDriver) Connect(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", d.url+"/health", nil)
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("vm not reachable: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("vm health check returned %d", resp.StatusCode)
	}
	return nil
}

func (d *vmDriver) Close() error {
	d.client.CloseIdleConnections()
	return nil
}

func (d *vmDriver) Write(ctx context.Context, key string, value float64) error {
	s := metrics.NewSet()
	s.GetOrCreateGauge(fmt.Sprintf(`benchmark_value{key="%s"}`, key), func() float64 {
		return value
	})

	var buf bytes.Buffer
	s.WritePrometheus(&buf)

	return d.importPrometheus(ctx, &buf)
}

func (d *vmDriver) WriteBatch(ctx context.Context, points []KeyedPoint) error {
	s := metrics.NewSet()

	for _, p := range points {
		s.GetOrCreateGauge(fmt.Sprintf(`benchmark_value{key="%s"}`, p.Key), func() float64 {
			return p.Value
		})
	}

	var buf bytes.Buffer
	s.WritePrometheus(&buf)

	return d.importPrometheus(ctx, &buf)
}

func (d *vmDriver) importPrometheus(ctx context.Context, body io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, "POST", d.url+"/api/v1/import/prometheus", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("vm import returned %d", resp.StatusCode)
	}
	return nil
}

func (d *vmDriver) Read(ctx context.Context, key string, lastX int) (int, error) {
	// Use a dedicated client for reads to avoid connection pool exhaustion from writes
	readClient := &http.Client{Timeout: 30 * time.Second}
	return d.readManyCtx(ctx, readClient, []string{key}, lastX)
}

func (d *vmDriver) readMany(ctx context.Context, keys []string, lastX int) (int, error) {
	return d.readManyCtx(ctx, d.client, keys, lastX)
}

func (d *vmDriver) readManyCtx(ctx context.Context, client *http.Client, keys []string, lastX int) (int, error) {
	// Use query_range to get last N points per key
	end := 1700001000
	start := end - lastX
	query := fmt.Sprintf(`benchmark_value{key=~"%s"}`, strings.Join(keys, "|"))
	req, err := http.NewRequestWithContext(ctx, "GET", d.url+"/api/v1/query_range", nil)
	if err != nil {
		return 0, err
	}

	q := req.URL.Query()
	q.Set("query", query)
	q.Set("start", fmt.Sprintf("%d", start))
	q.Set("end", fmt.Sprintf("%d", end))
	q.Set("step", "1")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var promResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string          `json:"resultType"`
			Result     json.RawMessage `json:"result"`
		} `json:"data"`
	}
	if err := sonic.Unmarshal(body, &promResp); err != nil {
		return 0, err
	}
	if promResp.Status != "success" {
		return 0, nil
	}

	type metricResult struct {
		Metric map[string]string `json:"metric"`
		Values [][]interface{}   `json:"values"`
	}

	var results []metricResult
	if err := sonic.Unmarshal(promResp.Data.Result, &results); err != nil {
		return 0, err
	}

	totalPoints := 0
	for _, r := range results {
		totalPoints += len(r.Values)
	}
	return totalPoints, nil
}

func (d *vmDriver) multiWrite(numPointsPerSensor, numSensors int) (success, failure uint64, elapsed time.Duration) {
	start := time.Now()

	s := metrics.NewSet()
	for i := 0; i < numSensors; i++ {
		key := fmt.Sprintf("benchmark_sensor_%d", i)
		for j := 0; j < numPointsPerSensor; j++ {
			val := float64(j) * 1.5
			s.GetOrCreateGauge(fmt.Sprintf(`benchmark_value{key="%s"}`, key), func() float64 {
				return val
			})
			success++
		}
	}

	var buf bytes.Buffer
	s.WritePrometheus(&buf)
	d.importPrometheus(context.Background(), &buf)

	elapsed = time.Since(start)
	return
}

// WritePipelined for VM uses Prometheus exposition format with multiple metric lines.
func (d *vmDriver) writePipelined(ctx context.Context, key string, values []float64) (int, error) {
	s := metrics.NewSet()

	for _, v := range values {
		s.GetOrCreateGauge(fmt.Sprintf(`benchmark_value{key="%s"}`, key), func() float64 {
			return v
		})
	}

	var buf bytes.Buffer
	s.WritePrometheus(&buf)

	if err := d.importPrometheus(ctx, &buf); err != nil {
		return 0, err
	}
	return len(values), nil
}
