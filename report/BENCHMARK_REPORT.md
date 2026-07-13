# Time Series Database Benchmark Report

**Generated:** 2026-07-13 23:15:31

## Overview

This report compares performance of the following time-series databases:

| Database | Version | Description |
|----------|---------|-------------|
| **GTSDB** | v1.0 | Custom Go time-series database (Hamster) |
| **InfluxDB** | v2.9.1 | Purpose-built time-series database |
| **VictoriaMetrics** | v1.147.0 | High-performance TSDB (Prometheus-compatible) |
| **NSQ** | v1.3.0 | Distributed messaging platform (pub/sub only) |

### Benchmark Configuration

| Parameter | Value |
|-----------|-------|
| Points per write | 5,000 |
| Sensors (multi-write) | 5 |
| Runs per benchmark | 3 |
| Warmup iterations | 300 |

### Resource Usage (Post-Benchmark)

![charts/resource_usage.png](charts/resource_usage.png)

## Overall Throughput Comparison

![charts/ops_per_sec.png](charts/ops_per_sec.png)

*Log scale. Higher bars indicate better throughput (ops/sec).*

## Performance Profile (Radar)

![charts/radar.png](charts/radar.png)

*Radar chart showing normalized performance across all benchmark types. 100 = best in class.*

## Write Benchmarks

### Sequential Single-Point Writes

![charts/write_latency.png](charts/write_latency.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 87.56206 ms | 3.466337 ms | 82.4158 ms | 92.4334 ms | 88.3566 ms | 92.4334 ms | 92.4334 ms | 34,261.4 |
| InfluxDB | 1.45 s | 21.897979 ms | 1.42 s | 1.48 s | 1.44 s | 1.48 s | 1.48 s | 2,071.5 |
| VM | 397.55074 ms | 11.091908 ms | 377.9507 ms | 411.5034 ms | 398.1386 ms | 411.5034 ms | 411.5034 ms | 7,546.2 |

### Pipelined Writes (GTSDB)

![charts/pipeline.png](charts/pipeline.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 30.3013 ms | 2.283717 ms | 27.2162 ms | 33.3332 ms | 29.4956 ms | 33.3332 ms | 33.3332 ms | 99,005.7 |
| InfluxDB | 260.87186 ms | 2.591997 ms | 257.2938 ms | 265.3323 ms | 260.7253 ms | 265.3323 ms | 265.3323 ms | 11,499.9 |
| VM | 220.81092 ms | 639 us | 220.2903 ms | 221.9518 ms | 220.3811 ms | 221.9518 ms | 221.9518 ms | 13,586.3 |

### Batch/Bulk Writes

![charts/batch_comparison.png](charts/batch_comparison.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 1.97308 ms | 607 us | 1.5356 ms | 3.128 ms | 1.5793 ms | 3.128 ms | 3.128 ms | 1,520,465.5 |
| InfluxDB | 10.77728 ms | 674 us | 9.6511 ms | 11.7689 ms | 10.8597 ms | 11.7689 ms | 11.7689 ms | 278,363.4 |
| VM | 134.31966 ms | 2.446659 ms | 130.5817 ms | 137.6402 ms | 134.4137 ms | 137.6402 ms | 137.6402 ms | 22,334.8 |

### Multi-Sensor Concurrent Writes

![charts/multi_write.png](charts/multi_write.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 2.3754 ms | 917 us | 1.5386 ms | 4.1639 ms | 2.0544 ms | 4.1639 ms | 4.1639 ms | 1,262,945.2 |
| InfluxDB | 8.2104 ms | 1.592228 ms | 7.139 ms | 11.3552 ms | 7.6916 ms | 11.3552 ms | 11.3552 ms | 365,390.2 |
| VM | 134.4183 ms | 912 us | 133.0237 ms | 135.5273 ms | 134.2405 ms | 135.5273 ms | 135.5273 ms | 22,318.4 |

## Read Benchmarks

![charts/read_comparison.png](charts/read_comparison.png)

### Single Read (Last 1 Point)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 190 us | 249 us | 0 s | 581 us | 0 s | 519 us | 554 us | 5,265.3 |
| InfluxDB | 8.083096 ms | 520 us | 7.1222 ms | 9.9547 ms | 8.1596 ms | 9.2913 ms | 9.7804 ms | 123.7 |
| VM | 96 us | 230 us | 0 s | 1.5359 ms | 0 s | 517 us | 1.0218 ms | 10,470.4 |

### Read-Many (5000 Reads)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 7.01546 ms | 7.723746 ms | 523 us | 18.0532 ms | 1.079 ms | 18.0532 ms | 18.0532 ms | 3,563,558.2 |
| InfluxDB | 11.05556 ms | 1.949366 ms | 9.36 ms | 14.8488 ms | 10.4869 ms | 14.8488 ms | 14.8488 ms | 2,261,305.6 |
| VM | 504 us | 777 us | 0 s | 2.0063 ms | 0 s | 2.0063 ms | 2.0063 ms | 49,634,688.7 |

## Pub/Sub Benchmark

![charts/pubsub.png](charts/pubsub.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 103.63792 ms | 3.2238 ms | 100.6632 ms | 109.4329 ms | 102.2095 ms | 109.4329 ms | 109.4329 ms | 9.6 |
| NSQ | 87.10394 ms | 1.084769 ms | 85.7426 ms | 88.8947 ms | 86.6248 ms | 88.8947 ms | 88.8947 ms | 11.5 |

## Key Findings

### Sequential Write Throughput

- #1 **GTSDB**: **34,261 ops/sec** (87.56206 ms)
- #2 **VM**: **7,546 ops/sec** (397.55074 ms)
- #3 **InfluxDB**: **2,071 ops/sec** (1.45 s)

### Batch Write Throughput

- #1 **GTSDB**: **1,520,465 ops/sec** (1.97308 ms)
- #2 **InfluxDB**: **278,363 ops/sec** (10.77728 ms)
- #3 **VM**: **22,335 ops/sec** (134.31966 ms)

### Multi-Sensor Write Throughput

- #1 **GTSDB**: **1,262,945 ops/sec** (2.3754 ms)
- #2 **InfluxDB**: **365,390 ops/sec** (8.2104 ms)
- #3 **VM**: **22,318 ops/sec** (134.4183 ms)

### Read Performance

- #1 **VM**: **10,470 ops/sec** (96 us)
- #2 **GTSDB**: **5,265 ops/sec** (190 us)
- #3 **InfluxDB**: **124 ops/sec** (8.083096 ms)

### Pub/Sub Latency

- #1 **NSQ**: **87.10394 ms** delivery latency
- #2 **GTSDB**: **103.63792 ms** delivery latency

## Head-to-Head Comparison

| Benchmark | Comparison | Ratio | Winner |
|-----------|------------|-------|--------|
| Write (seq) | GTSDB vs InfluxDB | 16.54x | **GTSDB** |
| Write (seq) | GTSDB vs VM | 4.54x | **GTSDB** |
| Pipeline Write | GTSDB vs InfluxDB | 8.61x | **GTSDB** |
| Pipeline Write | GTSDB vs VM | 7.29x | **GTSDB** |
| Batch Write | GTSDB vs InfluxDB | 5.46x | **GTSDB** |
| Batch Write | GTSDB vs VM | 68.08x | **GTSDB** |
| Read (single) | GTSDB vs InfluxDB | 42.56x | **GTSDB** |
| Read (single) | GTSDB vs VM | 1.99x | **VM** |
| Multi-Key Write | GTSDB vs InfluxDB | 3.46x | **GTSDB** |
| Multi-Key Write | GTSDB vs VM | 56.59x | **GTSDB** |
| Pub/Sub | GTSDB vs NSQ | 0.84x latency | **NSQ** |
| Multi-Key Read | GTSDB vs InfluxDB | 1.58x | **GTSDB** |
| Multi-Key Read | GTSDB vs VM | 13.93x | **VM** |
| Write (seq) | InfluxDB vs VM | 3.64x | **VM** |
| Pipeline Write | InfluxDB vs VM | 1.18x | **VM** |
| Batch Write | InfluxDB vs VM | 12.46x | **InfluxDB** |
| Read (single) | InfluxDB vs VM | 84.63x | **VM** |
| Multi-Key Write | InfluxDB vs VM | 16.37x | **InfluxDB** |
| Multi-Key Read | InfluxDB vs VM | 21.95x | **VM** |