package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"time"
)

const benchSensorKey = "benchmark_sensor"

// runWriteBenchmark performs sequential single-point writes with warmup and multiple runs.
func runWriteBenchmark(w Writer, key string, count, warmup, runs int) *BenchmarkResult {
	result := newBenchResult("Write (seq)", w.Name(), count)
	ctx := context.Background()

	for i := 0; i < warmup; i++ {
		w.Write(ctx, key, float64(i))
	}

	for run := 0; run < runs; run++ {
		start := time.Now()
		var success, failure uint64
		for i := 0; i < count; i++ {
			if err := w.Write(ctx, key, float64(i)); err == nil {
				success++
			} else {
				failure++
			}
		}
		result.addRun(time.Since(start), success, failure)
	}

	result.compute()
	return result
}

// runPipelinedWrite performs concurrent writes to the same key using all available parallelism.
// This tests each database's ability to handle write contention on a single timeseries,
// which is a meaningful scenario for all databases (unlike the old GTSDB-only TCP pipeline).
func runPipelinedWrite(w Writer, key string, count, runs int) *BenchmarkResult {
	concurrency := 8 // number of concurrent workers
	result := newBenchResult("Pipeline Write", w.Name(), count)
	ctx := context.Background()

	for run := 0; run < runs; run++ {
		var wg sync.WaitGroup
		opsPerWorker := count / concurrency
		var acc atomicAccumulator
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerIdx, startVal int) {
				defer wg.Done()
				var s, f uint64
				for j := 0; j < opsPerWorker; j++ {
					val := float64(startVal + j)
					if err := w.Write(ctx, key, val); err == nil {
						s++
					} else {
						f++
					}
				}
				acc.addSuccess(s)
				acc.addFailure(f)
			}(i, i*opsPerWorker)
		}
		wg.Wait()

		result.addRun(time.Since(start), acc.successCount(), acc.failureCount())
	}

	result.compute()
	return result
}

// runPipelinedWriteGTSDB uses GTSDB's TCP pipelining (fire-and-forget sends, then collect ACKs).
// This is strictly faster than the generic concurrent version for GTSDB because it
// avoids per-call mutex contention and TCP round-trip delays.
func runPipelinedWriteGTSDB(tcpAddr string, key string, count, runs int) *BenchmarkResult {
	result := newBenchResult("Pipeline Write", "GTSDB", count)

	for run := 0; run < runs; run++ {
		conn, err := net.Dial("tcp", tcpAddr)
		if err != nil {
			result.addRun(0, 0, uint64(count))
			continue
		}
		reader := bufio.NewReader(conn)

		start := time.Now()

		var success, failure uint64
		for i := 0; i < count; i++ {
			payload := fmt.Sprintf(`{"operation":"write","key":"%s","write":{"value":%f}}`, key, float64(i))
			if _, err := conn.Write(append([]byte(payload), '\n')); err != nil {
				failure++
				continue
			}
		}

		for i := 0; i < count; i++ {
			if _, err := reader.ReadBytes('\n'); err != nil {
				failure++
			} else {
				success++
			}
		}

		result.addRun(time.Since(start), success, failure)
		conn.Close()
	}

	result.compute()
	return result
}

// runBatchWrite performs bulk writes via batch API.
func runBatchWrite(w Writer, key string, count, runs int) *BenchmarkResult {
	result := newBenchResult("Batch Write", w.Name(), count)
	ctx := context.Background()

	for run := 0; run < runs; run++ {
		points := make([]KeyedPoint, count)
		ts := time.Now().Unix()
		for i := 0; i < count; i++ {
			points[i] = KeyedPoint{Key: key, Value: rand.Float64() * 100, Timestamp: ts + int64(i)}
		}

		start := time.Now()
		err := w.WriteBatch(ctx, points)
		if err == nil {
			result.addRun(time.Since(start), uint64(count), 0)
		} else {
			result.addRun(time.Since(start), 0, uint64(count))
		}
	}

	result.compute()
	return result
}

// runReadBenchmark performs individual read queries with warmup and multiple runs.
func runReadBenchmark(r Reader, key string, lastX, runs int) *BenchmarkResult {
	result := newBenchResult("Read (single)", r.Name(), 1)
	ctx := context.Background()

	r.Read(ctx, key, lastX)

	for run := 0; run < runs; run++ {
		start := time.Now()
		_, err := r.Read(ctx, key, lastX)
		if err == nil {
			result.addRun(time.Since(start), 1, 0)
		} else {
			result.addRun(time.Since(start), 0, 1)
		}
	}

	result.compute()
	return result
}

// runMultiWriteInflux performs concurrent multi-sensor writes via InfluxDB async WriteAPI.
func runMultiWriteInflux(d *influxDriver, numPointsPerSensor, numSensors, runs int) *BenchmarkResult {
	totalOps := numPointsPerSensor * numSensors
	result := newBenchResult("Multi-Key Write", d.Name(), totalOps)

	for run := 0; run < runs; run++ {
		s, f, d := d.multiWrite(numPointsPerSensor, numSensors)
		result.addRun(d, s, f)
	}
	result.compute()
	return result
}

// runMultiWriteVM performs concurrent multi-sensor writes via VictoriaMetrics.
func runMultiWriteVM(d *vmDriver, numPointsPerSensor, numSensors, runs int) *BenchmarkResult {
	totalOps := numPointsPerSensor * numSensors
	result := newBenchResult("Multi-Key Write", d.Name(), totalOps)

	for run := 0; run < runs; run++ {
		s, f, d := d.multiWrite(numPointsPerSensor, numSensors)
		result.addRun(d, s, f)
	}
	result.compute()
	return result
}

// runReadManyVM uses VM's range query to read N points per sensor.
func runReadManyVM(v *vmDriver, numSensors, pointsPerSensor, runs int) *BenchmarkResult {
	totalOps := numSensors * pointsPerSensor
	result := newBenchResult("Multi-Key Read", "VM", totalOps)
	ctx := context.Background()

	for run := 0; run < runs; run++ {
		keys := make([]string, numSensors)
		for i := 0; i < numSensors; i++ {
			keys[i] = fmt.Sprintf("bench_sensor_%d", i)
		}

		start := time.Now()
		count, err := v.readMany(ctx, keys, pointsPerSensor)
		if err == nil {
			// count = total data points returned across all sensors
			result.addRun(time.Since(start), uint64(count), 0)
		} else {
			result.addRun(time.Since(start), 0, uint64(count))
		}
	}
	result.compute()
	return result
}

// runReadManyInflux performs many individual Flux queries over HTTP keep-alive.
func runReadManyInflux(d *influxDriver, numSensors, pointsPerSensor, runs int) *BenchmarkResult {
	totalOps := numSensors * pointsPerSensor
	result := newBenchResult("Multi-Key Read", d.Name(), totalOps)
	ctx := context.Background()

	for run := 0; run < runs; run++ {
		start := time.Now()
		s, f := d.readMany(ctx, numSensors, pointsPerSensor)
		result.addRun(time.Since(start), s, f)
	}
	result.compute()
	return result
}

// runMultiReadGTSDB uses GTSDB's multi-read API via the shared connection.
func runMultiReadGTSDB(g *gtsdbDriver, numSensors, pointsPerSensor, runs int) *BenchmarkResult {
	totalOps := numSensors * pointsPerSensor
	result := newBenchResult("Multi-Key Read", "GTSDB", totalOps)
	ctx := context.Background()

	keys := make([]string, numSensors)
	for i := 0; i < numSensors; i++ {
		keys[i] = fmt.Sprintf("bench_sensor_%d", i)
	}

	for run := 0; run < runs; run++ {
		start := time.Now()
		counts, err := g.MultiRead(ctx, keys, pointsPerSensor)
		if err != nil {
			result.addRun(time.Since(start), 0, uint64(totalOps))
			continue
		}
		var success uint64
		for _, c := range counts {
			if c > 0 {
				success++
			}
		}
		result.addRun(time.Since(start), success, uint64(totalOps)-success)
	}
	result.compute()
	return result
}

// modelsDataPoint mirrors the GTSDB models.DataPoint for JSON unmarshal.
type modelsDataPoint struct {
	Key       string  `json:"key"`
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// runMultiWriteGTSDBBatch uses GTSDB's TCP batch-write API to write multiple sensors' data
// in a single TCP request (no HTTP overhead).
func runMultiWriteGTSDBBatch(g *gtsdbDriver, numPointsPerSensor, numSensors, runs int) *BenchmarkResult {
	totalOps := numPointsPerSensor * numSensors
	result := newBenchResult("Multi-Key Write", "GTSDB", totalOps)
	ctx := context.Background()

	for run := 0; run < runs; run++ {
		var allPoints []KeyedPoint
		ts := time.Now().Unix()

		for i := 0; i < numSensors; i++ {
			key := fmt.Sprintf("benchmark_sensor_%d", i)
			for j := 0; j < numPointsPerSensor; j++ {
				allPoints = append(allPoints, KeyedPoint{
					Key:       key,
					Value:     rand.Float64() * 100,
					Timestamp: ts + int64(i*numPointsPerSensor+j),
				})
			}
		}

		start := time.Now()
		var success, failure uint64
		// Split into batches of up to 10000 points (GTSDB limit)
		batchSize := 10000
		for b := 0; b < len(allPoints); b += batchSize {
			end := b + batchSize
			if end > len(allPoints) {
				end = len(allPoints)
			}
			err := g.writeBatchTCP(ctx, allPoints[b:end])
			if err == nil {
				success += uint64(end - b)
			} else {
				failure += uint64(end - b)
			}
		}
		result.addRun(time.Since(start), success, failure)
	}
	result.compute()
	return result
}

// runPubSubBenchmark runs pubsub latency benchmark.
func runPubSubBenchmark(p PubSuber, key string, count, runs int) *BenchmarkResult {
	result := newBenchResult("Pub/Sub", p.Name(), 1)

	for run := 0; run < runs; run++ {
		d, err := p.PubSub(context.Background(), key, count)
		if err == nil {
			result.addRun(d, 1, 0)
		} else {
			result.addRun(d, 0, 1)
		}
	}
	result.compute()
	return result
}

// preloadAndInit prepares data for read benchmarks.
func preloadAndInit(cfg *Config, g *gtsdbDriver, i *influxDriver, v *vmDriver) {
	if cfg.HasDB("gtsdb") && g != nil {
		fmt.Println("Pre-loading GTSDB...")
		g.initKeys(cfg.Sensors)
		time.Sleep(100 * time.Millisecond)
		g.preloadTCP(cfg.Sensors, 5000)
	}
	if cfg.HasDB("influx") && i != nil {
		fmt.Println("Pre-loading InfluxDB...")
		i.preload(cfg.Sensors, 5000)
		time.Sleep(500 * time.Millisecond) // wait for async flush to complete
	}
	if cfg.HasDB("vm") && v != nil {
		fmt.Println("Pre-loading VictoriaMetrics...")
		for s := 0; s < cfg.Sensors; s++ {
			points := make([]KeyedPoint, 5000)
			key := fmt.Sprintf("bench_sensor_%d", s)
			for j := 0; j < 5000; j++ {
				points[j] = KeyedPoint{Key: key, Value: rand.Float64() * 100, Timestamp: 1700000000 + int64(j)}
			}
			v.WriteBatch(context.Background(), points)
		}
		time.Sleep(200 * time.Millisecond) // wait for VM ingestion
	}
	fmt.Println("Pre-load done.")
}
