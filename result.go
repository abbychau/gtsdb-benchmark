package main

import (
	"math"
	"sort"
	"sync/atomic"
	"time"
)

type BenchmarkResult struct {
	Name          string
	DriverName    string
	OperationCount int
	successCount  uint64
	failureCount  uint64
	Durations     []time.Duration

	Min       time.Duration
	Max       time.Duration
	Mean      time.Duration
	StdDev    time.Duration
	P50       time.Duration
	P95       time.Duration
	P99       time.Duration
	OpsPerSec float64
	TotalOps  int64
}

type atomicAccumulator struct {
	success atomic.Uint64
	failure atomic.Uint64
}

func (a *atomicAccumulator) addSuccess(n uint64)  { a.success.Add(n) }
func (a *atomicAccumulator) addFailure(n uint64)  { a.failure.Add(n) }
func (a *atomicAccumulator) successCount() uint64 { return a.success.Load() }
func (a *atomicAccumulator) failureCount() uint64 { return a.failure.Load() }

func newBenchResult(name, driver string, opsPerRun int) *BenchmarkResult {
	return &BenchmarkResult{
		Name:          name,
		DriverName:    driver,
		OperationCount: opsPerRun,
	}
}

func (r *BenchmarkResult) addRun(d time.Duration, success, failure uint64) {
	r.Durations = append(r.Durations, d)
	r.successCount += success
	r.failureCount += failure
}

func (r *BenchmarkResult) compute() {
	if len(r.Durations) == 0 {
		return
	}

	sorted := make([]time.Duration, len(r.Durations))
	copy(sorted, r.Durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	n := len(sorted)
	r.Min = sorted[0]
	r.Max = sorted[n-1]

	var total time.Duration
	for _, d := range r.Durations {
		total += d
	}
	r.Mean = total / time.Duration(n)

	meanF := float64(r.Mean)
	var variance float64
	for _, d := range r.Durations {
		diff := float64(d) - meanF
		variance += diff * diff
	}
	variance /= float64(n)
	r.StdDev = time.Duration(math.Sqrt(variance))

	r.P50 = percentile(sorted, 0.50)
	r.P95 = percentile(sorted, 0.95)
	r.P99 = percentile(sorted, 0.99)

	if r.Mean > 0 {
		r.OpsPerSec = float64(r.OperationCount) / r.Mean.Seconds()
	}
	r.TotalOps = int64(r.OperationCount) * int64(n)
}

func (r *BenchmarkResult) SuccessRate() float64 {
	total := r.successCount + r.failureCount
	if total == 0 {
		return 0
	}
	return float64(r.successCount) / float64(total) * 100
}

func percentile(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	idx := int(math.Ceil(p*float64(len(durations)))) - 1
	if idx < 0 {
		idx = 0
	}
	return durations[idx]
}
