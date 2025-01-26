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
		Value float64 `json:"value"`
	} `json:"write"`
}

type GTSDBRead struct {
	Operation string `json:"operation"`

	Key  string `json:"key"`
	Read struct {
		StartTime    int64 `json:"start_timestamp,omitempty"`
		EndTime      int64 `json:"end_timestamp,omitempty"`
		Downsampling int   `json:"downsampling,omitempty"`
		LastX        int   `json:"lastx,omitempty"`
	} `json:"read"`
}

type GTSDBSubscribe struct {
	Operation string `json:"operation"`
	Key       string `json:"key"`
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
				Value float64 `json:"value"`
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
						Value float64 `json:"value"`
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

func benchmarkGTSDBPubSub(gtsdbAddress string, count int) BenchmarkResult {
	result := BenchmarkResult{}
	done := make(chan bool)

	conn, err := connectTCP(gtsdbAddress)
	if err != nil {
		return result
	}
	defer conn.Close()

	subConn, err := connectTCP(gtsdbAddress)
	if err != nil {
		return result
	}
	defer subConn.Close()

	// Set up subscription
	subReq := GTSDBSubscribe{
		Operation: "subscribe",
		Key:       "benchmark_sensor",
	}
	if err := writeToTCP(subConn, mustMarshal(subReq)); err != nil {
		return result
	}

	start := time.Now()
	var received atomic.Int64
	received.Store(0)

	// Start subscriber
	go func() {
		for {
			_, err := readFromTCP(subConn)
			if err != nil {
				return
			}

			newCount := received.Add(1)
			if newCount == int64(count) {
				result.Duration = time.Since(start)
				done <- true
				return
			}
		}
	}()

	// Wait for subscription to be established
	time.Sleep(1 * time.Second)

	// Start publisher
	go func() {
		margin := int(0.1 * float64(count))
		for i := 0; i < count+margin; i++ {
			pubReq := GTSDBWrite{
				Operation: "write",
				Key:       "benchmark_sensor",
				Write: struct {
					Value float64 `json:"value"`
				}{
					Value: float64(i),
				},
			}
			if err := writeToTCP(conn, mustMarshal(pubReq)); err != nil {
				return
			}
			readFromTCP(conn) // Read acknowledgment
		}
	}()

	<-done // Wait for benchmark to complete
	return result
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
