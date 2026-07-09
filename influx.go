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

func benchmarkInfluxMultiWrite(client influxdb2.Client, numPointsPerSensor int, numSensors int) BenchmarkResult {
	writeAPI := client.WriteAPIBlocking("abby", "abby")
	start := time.Now()
	result := BenchmarkResult{
		OperationCount: numPointsPerSensor * numSensors,
	}

	var wg sync.WaitGroup
	wg.Add(numSensors)

	for i := 0; i < numSensors; i++ {
		sensorID := fmt.Sprintf("benchmark_sensor_%d", i)
		go func(sid string) {
			defer wg.Done()

			for j := 0; j < numPointsPerSensor; j++ {
				p := influxdb2.NewPoint(
					"sensor_data",
					map[string]string{"sensor_id": sid},
					map[string]interface{}{"value": rand.Float64() * 100},
					time.Now(),
				)
				err := writeAPI.WritePoint(context.Background(), p)

				if err == nil {
					atomic.AddUint64(&result.SuccessCount, 1)
				} else {
					atomic.AddUint64(&result.FailureCount, 1)
				}
			}
		}(sensorID)
	}

	wg.Wait()

	result.Duration = time.Since(start)
	result.SuccessRate = float64(result.SuccessCount) / float64(result.OperationCount) * 100
	return result
}

func benchmarkInfluxWrite(client influxdb2.Client, sensorID string, numPoints int) BenchmarkResult {
	writeAPI := client.WriteAPIBlocking("abby", "abby")
	start := time.Now()
	result := BenchmarkResult{OperationCount: numPoints}

	for i := 0; i < numPoints; i++ {
		p := influxdb2.NewPoint(
			"sensor_data",
			map[string]string{"sensor_id": sensorID},
			map[string]interface{}{"value": float64(i)},
			time.Now(),
		)
		err := writeAPI.WritePoint(context.Background(), p)

		if err == nil {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	}

	result.Duration = time.Since(start)
	result.SuccessRate = float64(result.SuccessCount) / float64(result.OperationCount) * 100
	return result
}

func benchmarkInfluxRead(client influxdb2.Client, sensorID string) BenchmarkResult {
	queryAPI := client.QueryAPI("abby")
	start := time.Now()
	result := BenchmarkResult{OperationCount: 1}

	query := fmt.Sprintf(`from(bucket:"abby")
        |> range(start: -1h)
        |> filter(fn: (r) => r["sensor_id"] == "%s")
        |> limit(n:100)`, sensorID)

	_, err := queryAPI.Query(context.Background(), query)

	if err == nil {
		result.SuccessCount++
	} else {
		result.FailureCount++
		fmt.Println(err)
	}

	result.Duration = time.Since(start)
	result.SuccessRate = float64(result.SuccessCount) / float64(result.OperationCount) * 100
	return result
}

// ReadMany: 10,000 individual Flux queries via HTTP keep-alive
func benchmarkInfluxReadMany(influxURL, token, org, bucket string, numSensors, pointsPerSensor int) BenchmarkResult {
	start := time.Now()
	result := BenchmarkResult{OperationCount: numSensors * pointsPerSensor}

	client := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 100}}
	queryURL := fmt.Sprintf("%s/api/v2/query?org=%s", influxURL, org)

	for i := 0; i < numSensors; i++ {
		for j := 0; j < pointsPerSensor; j++ {
			flux := fmt.Sprintf(`from(bucket:"%s") |> range(start: 0) |> filter(fn: (r) => r._measurement == "sensor" and r.key == "sensor%d") |> last()`, bucket, i)
			req, _ := http.NewRequest("POST", queryURL, strings.NewReader(flux))
			req.Header.Set("Authorization", "Token "+token)
			req.Header.Set("Content-Type", "application/vnd.flux")
			req.Header.Set("Accept", "application/csv")
			resp, err := client.Do(req)
			if err == nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				result.SuccessCount++
			} else {
				result.FailureCount++
			}
		}
	}

	result.Duration = time.Since(start)
	result.SuccessRate = float64(result.SuccessCount) / float64(result.OperationCount) * 100
	return result
}

// PreloadInfluxDB loads data via line protocol
func PreloadInfluxDB(influxURL, token, org, bucket string, numSensors, pointsPerSensor int) {
	writeURL := fmt.Sprintf("%s/api/v2/write?org=%s&bucket=%s", influxURL, org, bucket)
	client := &http.Client{}
	for i := 0; i < numSensors; i++ {
		var lines []string
		for j := 0; j < pointsPerSensor; j++ {
			lines = append(lines, fmt.Sprintf("sensor,key=sensor%d value=%f %d", i, float64(j)*1.5, 1700000000+int64(j)))
		}
		req, _ := http.NewRequest("POST", writeURL, strings.NewReader(strings.Join(lines, "\n")))
		req.Header.Set("Authorization", "Token "+token)
		req.Header.Set("Content-Type", "text/plain")
		resp, _ := client.Do(req)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
