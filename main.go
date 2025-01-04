package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type GTSDBWrite struct {
	Operation string `json:"operation"`
	Key       string `json:"key"`
	Write     struct {
		Value float64 `json:"Value"`
	} `json:"Write"`
}

type GTSDBRead struct {
	Operation string `json:"operation"`

	Key  string `json:"key"`
	Read struct {
		StartTime    int64 `json:"start_timestamp,omitempty"`
		EndTime      int64 `json:"end_timestamp,omitempty"`
		Downsampling int   `json:"downsampling,omitempty"`
		LastX        int   `json:"lastx,omitempty"`
	} `json:"Read"`
}
type BenchmarkResult struct {
	OperationCount int
	SuccessCount   uint64
	FailureCount   uint64
	Duration       time.Duration
	SuccessRate    float64
}

func connectTCP(address string) (net.Conn, error) {
	return net.Dial("tcp", address)
}

func writeToTCP(conn net.Conn, data []byte) error {
	_, err := conn.Write(append(data, '\n'))
	return err
}

func readFromTCP(conn net.Conn) ([]byte, error) {
	reader := bufio.NewReader(conn)
	return reader.ReadBytes('\n')
}

func benchmarkGTSDBWrite(address string, sensorID string, numPoints int) BenchmarkResult {
	start := time.Now()
	result := BenchmarkResult{OperationCount: numPoints}

	conn, err := connectTCP(address)
	if err != nil {
		result.FailureCount = uint64(numPoints)
		return result
	}
	defer conn.Close()

	for i := 0; i < numPoints; i++ {
		writeReq := GTSDBWrite{
			Operation: "write",

			Key: sensorID,
			Write: struct {
				Value float64 `json:"Value"`
			}{
				Value: float64(i),
			},
		}

		payload, _ := json.Marshal(writeReq)
		err := writeToTCP(conn, payload)

		if err == nil {
			_, readErr := readFromTCP(conn)
			if readErr == nil {
				result.SuccessCount++
			} else {
				result.FailureCount++
			}
		} else {
			result.FailureCount++
		}
	}

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

func benchmarkGTSDBRead(address string, sensorID string) BenchmarkResult {
	start := time.Now()
	result := BenchmarkResult{OperationCount: 1}

	conn, err := connectTCP(address)
	if err != nil {
		result.FailureCount = 1
		return result
	}
	defer conn.Close()

	readReq := GTSDBRead{
		Operation: "read",

		Key: sensorID,
		Read: struct {
			StartTime    int64 `json:"start_timestamp,omitempty"`
			EndTime      int64 `json:"end_timestamp,omitempty"`
			Downsampling int   `json:"downsampling,omitempty"`
			LastX        int   `json:"lastx,omitempty"`
		}{
			LastX: 100,
		},
	}

	payload, _ := json.Marshal(readReq)
	err = writeToTCP(conn, payload)

	if err == nil {
		_, readErr := readFromTCP(conn)
		if readErr == nil {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	} else {
		result.FailureCount++
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

func main() {
	// Initialize clients
	gtsdbAddress := "localhost:5555"
	influxURL := "http://localhost:8086"
	influxToken := "0sOuyNL1DM4fxLpYlRxvxaKOyjCs3M1VuUICdx81YaNwQLzsrmAKYoJLqUiKGICV4x3SaQFOq12eclj4IB64Qg=="

	influxClient := influxdb2.NewClient(influxURL, influxToken)

	defer influxClient.Close()

	numPoints := 10000
	sensorID := "benchmark_sensor"

	// Run benchmarks
	gtsdbWriteResult := benchmarkGTSDBWrite(gtsdbAddress, sensorID, numPoints)
	influxWriteResult := benchmarkInfluxWrite(influxClient, sensorID, numPoints)
	gtsdbReadResult := benchmarkGTSDBRead(gtsdbAddress, sensorID)
	influxReadResult := benchmarkInfluxRead(influxClient, sensorID)

	// Print results
	fmt.Printf("Write Performance:\n")
	fmt.Printf("GTSDB: %v (Success Rate: %.2f%%)\n", gtsdbWriteResult.Duration, gtsdbWriteResult.SuccessRate)
	fmt.Printf("InfluxDB: %v (Success Rate: %.2f%%)\n", influxWriteResult.Duration, influxWriteResult.SuccessRate)

	fmt.Printf("\nRead Performance:\n")
	fmt.Printf("GTSDB: %v (Success Rate: %.2f%%)\n", gtsdbReadResult.Duration, gtsdbReadResult.SuccessRate)
	fmt.Printf("InfluxDB: %v (Success Rate: %.2f%%)\n", influxReadResult.Duration, influxReadResult.SuccessRate)

	// Multi-write benchmark
	numPointsPerSensor := 1000
	numSensors := 10

	gtsdbMultiWriteResult := benchmarkGTSDBMultiWrite(gtsdbAddress, numPointsPerSensor, numSensors)
	influxMultiWriteResult := benchmarkInfluxMultiWrite(influxClient, numPointsPerSensor, numSensors)

	fmt.Printf("\nMulti-Write Performance:\n")
	fmt.Printf("GTSDB: %v (Success Rate: %.2f%%)\n", gtsdbMultiWriteResult.Duration, gtsdbMultiWriteResult.SuccessRate)
	fmt.Printf("InfluxDB: %v (Success Rate: %.2f%%)\n", influxMultiWriteResult.Duration, influxMultiWriteResult.SuccessRate)

}

func benchmarkGTSDBMultiWrite(address string, numPointsPerSensor int, numSensors int) BenchmarkResult {
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

			conn, err := connectTCP(address)
			if err != nil {
				atomic.AddUint64(&result.FailureCount, uint64(numPointsPerSensor))
				return
			}
			defer conn.Close()

			for j := 0; j < numPointsPerSensor; j++ {
				data := GTSDBWrite{
					Operation: "write",

					Key: sid,
					Write: struct {
						Value float64 `json:"Value"`
					}{
						Value: rand.Float64() * 100,
					},
				}

				jsonData, _ := json.Marshal(data)
				err := writeToTCP(conn, jsonData)

				if err == nil {
					_, readErr := readFromTCP(conn)
					if readErr == nil {
						atomic.AddUint64(&result.SuccessCount, 1)
					} else {
						atomic.AddUint64(&result.FailureCount, 1)
					}
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
