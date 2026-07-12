package main

import (
	"testing"
	"time"
)

func TestBenchmarkResultCompute(t *testing.T) {
	r := newBenchResult("test", "testdb", 100)
	d1 := 100 * time.Millisecond
	d2 := 200 * time.Millisecond
	d3 := 150 * time.Millisecond

	r.addRun(d1, 100, 0)
	r.addRun(d2, 100, 0)
	r.addRun(d3, 100, 0)

	r.compute()

	if r.Min != d1 {
		t.Errorf("expected min %v, got %v", d1, r.Min)
	}
	if r.Max != d2 {
		t.Errorf("expected max %v, got %v", d2, r.Max)
	}
	if r.Mean != 150*time.Millisecond {
		t.Errorf("expected mean 150ms, got %v", r.Mean)
	}
	if r.TotalOps != 300 {
		t.Errorf("expected totalOps 300, got %d", r.TotalOps)
	}
	if r.SuccessRate() != 100.0 {
		t.Errorf("expected success rate 100%%, got %.2f", r.SuccessRate())
	}
}

func TestBenchmarkResultSingleRun(t *testing.T) {
	r := newBenchResult("test", "testdb", 1)
	r.addRun(10*time.Millisecond, 1, 0)
	r.compute()

	if r.Min != 10*time.Millisecond {
		t.Errorf("expected min 10ms, got %v", r.Min)
	}
	if r.Mean != 10*time.Millisecond {
		t.Errorf("expected mean 10ms, got %v", r.Mean)
	}
	if r.OpsPerSec != 100.0 {
		t.Errorf("expected ops/sec 100, got %.2f", r.OpsPerSec)
	}
}

func TestBenchmarkResultEmpty(t *testing.T) {
	r := newBenchResult("test", "testdb", 100)
	r.compute()

	if r.Mean != 0 {
		t.Errorf("expected mean 0, got %v", r.Mean)
	}
	if r.OpsPerSec != 0 {
		t.Errorf("expected ops/sec 0, got %.2f", r.OpsPerSec)
	}
}

func TestPercentile(t *testing.T) {
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}

	p50 := percentile(durations, 0.50)
	if p50 != 30*time.Millisecond {
		t.Errorf("expected p50 30ms, got %v", p50)
	}

	p95 := percentile(durations, 0.95)
	if p95 != 50*time.Millisecond {
		t.Errorf("expected p95 50ms, got %v", p95)
	}

	p99 := percentile(durations, 0.99)
	if p99 != 50*time.Millisecond {
		t.Errorf("expected p99 50ms, got %v", p99)
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "a") {
		t.Error("expected contains to return true for 'a'")
	}
	if contains(slice, "d") {
		t.Error("expected contains to return false for 'd'")
	}
}

func TestParseCSV(t *testing.T) {
	result := parseCSV("gtsdb,influx,nsq")
	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if result[0] != "gtsdb" || result[1] != "influx" || result[2] != "nsq" {
		t.Errorf("unexpected parse result: %v", result)
	}

	result2 := parseCSV("  gtsdb  ,  influx  ")
	if len(result2) != 2 || result2[0] != "gtsdb" || result2[1] != "influx" {
		t.Errorf("unexpected trimmed parse result: %v", result2)
	}
}

func TestReportEntry(t *testing.T) {
	r := newBenchResult("write", "GTSDB", 1000)
	r.addRun(100*time.Millisecond, 1000, 0)
	r.compute()

	entry := newReportEntry(r)

	if entry.Name != "write" {
		t.Errorf("expected name 'write', got %s", entry.Name)
	}
	if entry.Driver != "GTSDB" {
		t.Errorf("expected driver 'GTSDB', got %s", entry.Driver)
	}
	if entry.Runs != 1 {
		t.Errorf("expected 1 run, got %d", entry.Runs)
	}
	if entry.Ops != 1000 {
		t.Errorf("expected 1000 ops, got %d", entry.Ops)
	}
	if entry.SuccessRate != 100.0 {
		t.Errorf("expected 100%% success, got %.2f", entry.SuccessRate)
	}
}
