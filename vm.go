package main

// import (
// 	"fmt"
// 	"math/rand/v2"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	"github.com/VictoriaMetrics/metrics"
// )

// func benchmarkVMMultiWrite(set *metrics.Set, numPointsPerSensor int, numSensors int) BenchmarkResult {
// 	start := time.Now()
// 	result := BenchmarkResult{
// 		OperationCount: numPointsPerSensor * numSensors,
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(numSensors)

// 	for i := 0; i < numSensors; i++ {
// 		sensorID := fmt.Sprintf("benchmark_sensor_%d", i)
// 		go func(sid string) {
// 			defer wg.Done()

// 			for j := 0; j < numPointsPerSensor; j++ {
// 				metric := fmt.Sprintf(`sensor_data{sensor_id="%s"} %f`, sid, rand.Float64()*100)
// 				err := set.Set(metric)

// 				if err == nil {
// 					atomic.AddUint64(&result.SuccessCount, 1)
// 				} else {
// 					atomic.AddUint64(&result.FailureCount, 1)
// 				}
// 			}
// 		}(sensorID)
// 	}

// 	wg.Wait()

// 	result.Duration = time.Since(start)
// 	result.SuccessRate = float64(result.SuccessCount) / float64(result.OperationCount) * 100
// 	return result
// }

// func benchmarkVMWrite(set *metrics.Set, sensorID string, numPoints int) BenchmarkResult {
// 	start := time.Now()
// 	result := BenchmarkResult{OperationCount: numPoints}

// 	for i := 0; i < numPoints; i++ {
// 		metric := fmt.Sprintf(`sensor_data{sensor_id="%s"} %f`, sensorID, float64(i))
// 		err := set.Set(metric)

// 		if err == nil {
// 			result.SuccessCount++
// 		} else {
// 			result.FailureCount++
// 		}
// 	}

// 	result.Duration = time.Since(start)
// 	result.SuccessRate = float64(result.SuccessCount) / float64(result.OperationCount) * 100
// 	return result
// }

// func benchmarkVMRead(set *metrics.Set, sensorID string) BenchmarkResult {
// 	start := time.Now()
// 	result := BenchmarkResult{OperationCount: 1}

// 	// VictoriaMetrics metrics library primarily focuses on write operations
// 	// For reading metrics, you would typically use their HTTP API or Prometheus API
// 	// This is a simplified example that just checks if the metric exists
// 	metric := fmt.Sprintf(`sensor_data{sensor_id="%s"}`, sensorID)
// 	if set.GetOrCreateCounter(metric) != nil {
// 		result.SuccessCount++
// 	} else {
// 		result.FailureCount++
// 	}

// 	result.Duration = time.Since(start)
// 	result.SuccessRate = float64(result.SuccessCount) / float64(result.OperationCount) * 100
// 	return result
// }
