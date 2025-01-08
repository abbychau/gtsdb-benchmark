package main

import (
	"context"
	"fmt"
	"math/rand/v2"
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
