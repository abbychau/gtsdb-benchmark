package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

// MemorySample represents a single memory measurement at a point in time.
type MemorySample struct {
	T       float64 `json:"t"`        // Elapsed time in seconds
	AllocMB float64 `json:"alloc_mb"` // Go runtime Alloc (heap in use)
	SysMB   float64 `json:"sys_mb"`   // Go runtime Sys (OS memory retained)
	GCDone  int32   `json:"gc_done"`  // Number of completed GC cycles
}

type MemBenchOutput struct {
	Samples     []MemorySample `json:"samples"`
	TotalWrites int64          `json:"total_writes"`
	TotalReads  int64          `json:"total_reads"`
}

func sampleMemory() (AllocMB, SysMB float64, GCDone int32) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / (1024 * 1024),
		float64(m.Sys) / (1024 * 1024),
		int32(m.NumGC)
}

// MemoryBenchConfig configures the memory-over-time benchmark.
type MemoryBenchConfig struct {
	GTSDBAddr   string
	Duration    time.Duration
	Interval    time.Duration
	Keys        int
	Concurrency int // number of parallel pipelined connections
	Pipeline    int // number of writes to pipeline per batch
	BatchSize   int // number of keys per batch-write (0 = single writes)
}

func pipelinedWorker(tcpAddr string, keys []string, done <-chan struct{}, writeCount, readCount *atomic.Int64) {
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	buf := make([]byte, 0, 4096)

	for {
		select {
		case <-done:
			return
		default:
		}

		// Pipeline: write N payloads without waiting
		n := 100
		buf = buf[:0]
		for i := 0; i < n; i++ {
			key := keys[rand.IntN(len(keys))]
			buf = append(buf, []byte(fmt.Sprintf(
				`{"operation":"write","key":"%s","write":{"value":%f}}`+"\n",
				key, rand.Float64()*100,
			))...)
		}
		if _, err := conn.Write(buf); err != nil {
			return
		}
		// Drain responses
		for i := 0; i < n; i++ {
			if _, err := reader.ReadBytes('\n'); err != nil {
				return
			}
		}
		writeCount.Add(int64(n))

		// Occasional batch read
		if rand.IntN(20) == 0 {
			key := keys[rand.IntN(len(keys))]
			rp := fmt.Sprintf(
				`{"operation":"read","key":"%s","read":{"lastx":1000}}`+"\n",
				key,
			)
			if _, err := conn.Write([]byte(rp)); err != nil {
				return
			}
			if _, err := reader.ReadBytes('\n'); err != nil {
				return
			}
			readCount.Add(1)
		}
	}
}

func runMemoryBench(cfg MemoryBenchConfig) ([]MemorySample, int64, int64, error) {
	fmt.Fprintf(os.Stderr, "Connecting to GTSDB at %s...\n", cfg.GTSDBAddr)
	fmt.Fprintf(os.Stderr, "Workload: %d concurrent pipelines × %d pipeline depth, %d keys, %d batch size\n",
		cfg.Concurrency, cfg.Pipeline, cfg.Keys, cfg.BatchSize)
	fmt.Fprintf(os.Stderr, "Sampling memory every %v for %v...\n", cfg.Interval, cfg.Duration)

	keys := make([]string, cfg.Keys)
	for i := range keys {
		keys[i] = fmt.Sprintf("mem_stress_%d", i)
	}

	var samples []MemorySample
	startTime := time.Now()

	alloc, sys, gc := sampleMemory()
	samples = append(samples, MemorySample{T: 0, AllocMB: alloc, SysMB: sys, GCDone: gc})

	var writeCount, readCount atomic.Int64
	done := make(chan struct{})

	// Start concurrent pipelined workers
	for i := 0; i < cfg.Concurrency; i++ {
		go pipelinedWorker(cfg.GTSDBAddr, keys, done, &writeCount, &readCount)
	}

	sampleTicker := time.NewTicker(cfg.Interval)
	defer sampleTicker.Stop()

	sampleCount := int(cfg.Duration / cfg.Interval)
	for i := 0; i < sampleCount; i++ {
		<-sampleTicker.C
		alloc, sys, gc := sampleMemory()
		elapsed := time.Since(startTime).Seconds()
		samples = append(samples, MemorySample{T: elapsed, AllocMB: alloc, SysMB: sys, GCDone: gc})
	}

	close(done)

	return samples, writeCount.Load(), readCount.Load(), nil
}

func main() {
	cfg := MemoryBenchConfig{
		GTSDBAddr:   "localhost:5555",
		Duration:    30 * time.Second,
		Interval:    1 * time.Second,
		Keys:        50,  // 10× more keys for wider load
		Concurrency: 8,   // 8 parallel pipelined connections
		Pipeline:    100, // 100 writes per pipeline burst
		BatchSize:   0,   // use single writes in pipeline
	}
	if len(os.Args) > 1 {
		cfg.GTSDBAddr = os.Args[1]
	}
	if len(os.Args) > 2 {
		d, err := time.ParseDuration(os.Args[2])
		if err == nil {
			cfg.Duration = d
		}
	}

	samples, totalWrites, totalReads, err := runMemoryBench(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	output := struct {
		Samples     []MemorySample `json:"samples"`
		TotalWrites int64          `json:"total_writes"`
		TotalReads  int64          `json:"total_reads"`
	}{
		Samples:     samples,
		TotalWrites: totalWrites,
		TotalReads:  totalReads,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Output error: %v\n", err)
		os.Exit(1)
	}
}
