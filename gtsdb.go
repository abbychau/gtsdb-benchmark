package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
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
