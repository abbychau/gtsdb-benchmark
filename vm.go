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
	return d.importJSON(ctx, fmt.Sprintf(
		`{"metric":{"__name__":"benchmark_value","key":"%s"},"values":[%f],"timestamps":[%d]}`+"\n",
		key, value, time.Now().Unix()))
}

func (d *vmDriver) WriteBatch(ctx context.Context, points []KeyedPoint) error {
	groups := make(map[string][]KeyedPoint)
	for _, p := range points {
		groups[p.Key] = append(groups[p.Key], p)
	}
	var buf bytes.Buffer
	for key, pts := range groups {
		buf.WriteString(`{"metric":{"__name__":"benchmark_value","key":"`)
		buf.WriteString(key)
		buf.WriteString(`"},"values":[`)
		for i, p := range pts {
			if i > 0 {
				buf.WriteByte(',')
			}
			fmt.Fprintf(&buf, "%f", p.Value)
		}
		buf.WriteString(`],"timestamps":[`)
		for i, p := range pts {
			if i > 0 {
				buf.WriteByte(',')
			}
			fmt.Fprintf(&buf, "%d", p.Timestamp)
		}
		buf.WriteString("]}\n")
	}
	return d.importJSON(ctx, buf.String())
}

func (d *vmDriver) importJSON(ctx context.Context, body string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", d.url+"/api/v1/import", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
	readClient := &http.Client{Timeout: 30 * time.Second}
	return d.readManyCtx(ctx, readClient, []string{key}, lastX, 0)
}

func (d *vmDriver) readMany(ctx context.Context, keys []string, lastX int) (int, error) {
	return d.readManyCtx(ctx, d.client, keys, lastX, 0)
}

func (d *vmDriver) readManyCtx(ctx context.Context, client *http.Client, keys []string, lastX int, _ int64) (int, error) {
	// Query returns exactly lastX points: step=1 over lastX seconds
	end := int64(1700005000)
	start := end - int64(lastX)
	if start < 1700000000 {
		start = 1700000000
	}
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
	now := time.Now().Unix()

	var buf bytes.Buffer
	for i := 0; i < numSensors; i++ {
		key := fmt.Sprintf("benchmark_sensor_%d", i)
		buf.WriteString(`{"metric":{"__name__":"benchmark_value","key":"`)
		buf.WriteString(key)
		buf.WriteString(`"},"values":[`)
		for j := 0; j < numPointsPerSensor; j++ {
			if j > 0 {
				buf.WriteByte(',')
			}
			fmt.Fprintf(&buf, "%f", float64(j)*1.5)
		}
		buf.WriteString(`],"timestamps":[`)
		for j := 0; j < numPointsPerSensor; j++ {
			if j > 0 {
				buf.WriteByte(',')
			}
			fmt.Fprintf(&buf, "%d", now+int64(j))
		}
		buf.WriteString("]}\n")
		success += uint64(numPointsPerSensor)
	}

	d.importJSON(context.Background(), buf.String())
	elapsed = time.Since(start)
	return
}

func (d *vmDriver) writePipelined(ctx context.Context, key string, values []float64) (int, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"metric":{"__name__":"benchmark_value","key":"`)
	buf.WriteString(key)
	buf.WriteString(`"},"values":[`)
	now := time.Now().Unix()
	for i, v := range values {
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(&buf, "%f", v)
	}
	buf.WriteString(`],"timestamps":[`)
	for i := range values {
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(&buf, "%d", now+int64(i))
	}
	buf.WriteString("]}\n")
	if err := d.importJSON(ctx, buf.String()); err != nil {
		return 0, err
	}
	return len(values), nil
}
