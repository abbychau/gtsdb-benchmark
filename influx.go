package main

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type influxDriver struct {
	url    string
	token  string
	org    string
	bucket string
	client influxdb2.Client
}

func newInfluxDriver(url, token, org, bucket string) *influxDriver {
	return &influxDriver{
		url:    url,
		token:  token,
		org:    org,
		bucket: bucket,
	}
}

func (d *influxDriver) Name() string { return "InfluxDB" }

func (d *influxDriver) Connect(ctx context.Context) error {
	d.client = influxdb2.NewClientWithOptions(d.url, d.token,
		influxdb2.DefaultOptions().SetBatchSize(5000).SetFlushInterval(1000))
	_, err := d.client.Ready(ctx)
	if err != nil {
		return fmt.Errorf("influxdb not ready: %w", err)
	}
	return nil
}

func (d *influxDriver) Close() error {
	if d.client != nil {
		d.client.Close()
	}
	return nil
}

func (d *influxDriver) Write(ctx context.Context, key string, value float64) error {
	writeAPI := d.client.WriteAPIBlocking(d.org, d.bucket)
	p := influxdb2.NewPoint(
		"sensor_data",
		map[string]string{"sensor_id": key},
		map[string]interface{}{"value": value},
		time.Now(),
	)
	return writeAPI.WritePoint(ctx, p)
}

func (d *influxDriver) WriteBatch(ctx context.Context, points []KeyedPoint) error {
	writeAPI := d.client.WriteAPI(d.org, d.bucket)
	for _, point := range points {
		p := influxdb2.NewPoint(
			"sensor_data",
			map[string]string{"sensor_id": point.Key},
			map[string]interface{}{"value": point.Value},
			time.Unix(point.Timestamp, 0),
		)
		writeAPI.WritePoint(p)
	}
	writeAPI.Flush()
	return nil
}

func (d *influxDriver) Read(ctx context.Context, key string, lastX int) (int, error) {
	queryAPI := d.client.QueryAPI(d.org)
	query := fmt.Sprintf(`from(bucket:"%s")
	|> range(start: -1h)
	|> filter(fn: (r) => r["sensor_id"] == "%s")
	|> limit(n:%d)`, d.bucket, key, lastX)

	records, err := queryAPI.Query(ctx, query)
	if err != nil {
		return 0, err
	}

	count := 0
	for records.Next() {
		count++
	}
	return count, nil
}

func (d *influxDriver) readMany(ctx context.Context, numSensors, pointsPerSensor int) (success, failure uint64) {
	httpClient := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 100}}
	queryURL := fmt.Sprintf("%s/api/v2/query?org=%s", d.url, d.org)

	var wg sync.WaitGroup
	wg.Add(numSensors)

	for i := 0; i < numSensors; i++ {
		go func(sensorIdx int) {
			defer wg.Done()
			// Read last N points per sensor in a single query
			flux := fmt.Sprintf(`from(bucket:"%s") |> range(start: 0) |> filter(fn: (r) => r._measurement == "sensor" and r.key == "sensor%d") |> sort(columns: ["_time"], desc: true) |> limit(n:%d)`, d.bucket, sensorIdx, pointsPerSensor)
			req, _ := http.NewRequestWithContext(ctx, "POST", queryURL, strings.NewReader(flux))
			req.Header.Set("Authorization", "Token "+d.token)
			req.Header.Set("Content-Type", "application/vnd.flux")
			req.Header.Set("Accept", "application/csv")
			resp, err := httpClient.Do(req)
			if err == nil {
				// Count rows in CSV response minus header
				body, _ := io.ReadAll(resp.Body)
				lines := strings.Split(string(body), "\n")
				count := 0
				for _, line := range lines {
					if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "result") {
						count++
					}
				}
				resp.Body.Close()
				atomic.AddUint64(&success, uint64(count))
			} else {
				atomic.AddUint64(&failure, uint64(pointsPerSensor))
			}
		}(i)
	}
	wg.Wait()
	return
}

func (d *influxDriver) preload(numSensors, pointsPerSensor int) error {
	writeAPI := d.client.WriteAPI(d.org, d.bucket)

	for i := 0; i < numSensors; i++ {
		for j := 0; j < pointsPerSensor; j++ {
			p := influxdb2.NewPoint(
				"sensor",
				map[string]string{"key": fmt.Sprintf("sensor%d", i)},
				map[string]interface{}{"value": rand.Float64() * 100},
				time.Unix(int64(1700000000+j), 0),
			)
			writeAPI.WritePoint(p)
		}
	}

	writeAPI.Flush()
	return nil
}

// multiWrite performs concurrent writes across multiple sensors.
func (d *influxDriver) multiWrite(numPointsPerSensor, numSensors int) (success, failure uint64, elapsed time.Duration) {
	writeAPI := d.client.WriteAPI(d.org, d.bucket)
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(numSensors)

	for i := 0; i < numSensors; i++ {
		go func(sid string) {
			defer wg.Done()
			for j := 0; j < numPointsPerSensor; j++ {
				p := influxdb2.NewPoint(
					"sensor_data",
					map[string]string{"sensor_id": sid},
					map[string]interface{}{"value": rand.Float64() * 100},
					time.Now(),
				)
				writeAPI.WritePoint(p)
				atomic.AddUint64(&success, 1)
			}
		}(fmt.Sprintf("benchmark_sensor_%d", i))
	}

	wg.Wait()
	writeAPI.Flush()
	elapsed = time.Since(start)
	return
}
