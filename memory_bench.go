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
	GTSDBAddr string
	Duration  time.Duration
	Interval  time.Duration
	Keys      int
}

func runMemoryBench(cfg MemoryBenchConfig) ([]MemorySample, int64, int64, error) {
	conn, err := net.Dial("tcp", cfg.GTSDBAddr)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("connect to GTSDB: %w", err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	fmt.Fprintf(os.Stderr, "Connected to GTSDB at %s\n", cfg.GTSDBAddr)
	fmt.Fprintf(os.Stderr, "Sampling memory every %v for %v...\n", cfg.Interval, cfg.Duration)

	keys := make([]string, cfg.Keys)
	for i := range keys {
		keys[i] = fmt.Sprintf("mem_bench_sensor_%d", i)
	}

	var samples []MemorySample
	startTime := time.Now()
	endTime := startTime.Add(cfg.Duration)

	alloc, sys, gc := sampleMemory()
	samples = append(samples, MemorySample{T: 0, AllocMB: alloc, SysMB: sys, GCDone: gc})

	var writeCount, readCount atomic.Int64

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if time.Now().After(endTime) {
				return
			}
			key := keys[rand.IntN(len(keys))]
			payload := fmt.Sprintf(
				`{"operation":"write","key":"%s","write":{"value":%f}}`+"\n",
				key, rand.Float64()*100,
			)
			if _, err := conn.Write([]byte(payload)); err != nil {
				return
			}
			if _, err := reader.ReadBytes('\n'); err != nil {
				return
			}
			writeCount.Add(1)
			if rand.IntN(10) == 0 {
				rp := fmt.Sprintf(
					`{"operation":"read","key":"%s","read":{"lastx":100}}`+"\n",
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
	}()

	sampleTicker := time.NewTicker(cfg.Interval)
	defer sampleTicker.Stop()

	sampleCount := int(cfg.Duration / cfg.Interval)
	for i := 0; i < sampleCount; i++ {
		<-sampleTicker.C
		alloc, sys, gc := sampleMemory()
		elapsed := time.Since(startTime).Seconds()
		samples = append(samples, MemorySample{T: elapsed, AllocMB: alloc, SysMB: sys, GCDone: gc})
	}

	return samples, writeCount.Load(), readCount.Load(), nil
}

func main() {
	cfg := MemoryBenchConfig{
		GTSDBAddr: "localhost:5555",
		Duration:  30 * time.Second,
		Interval:  1 * time.Second,
		Keys:      5,
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
