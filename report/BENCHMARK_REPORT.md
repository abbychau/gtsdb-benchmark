# Time Series Database Benchmark Report

**Generated:** 2026-07-13 19:05:35

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
| GTSDB | 83.6477 ms | 2.3063 ms | 81.3414 ms | 85.954 ms | 81.3414 ms | 85.954 ms | 85.954 ms | 35,864.7 |
| InfluxDB | 1.45 s | 10.19085 ms | 1.44 s | 1.46 s | 1.44 s | 1.46 s | 1.46 s | 2,063.4 |
| VM | 157.234 ms | 2.1358 ms | 155.0982 ms | 159.3698 ms | 155.0982 ms | 159.3698 ms | 159.3698 ms | 19,079.8 |

### Pipelined Writes (GTSDB)

![charts/pipeline.png](charts/pipeline.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 27.90925 ms | 471 us | 27.4385 ms | 28.38 ms | 27.4385 ms | 28.38 ms | 28.38 ms | 107,491.2 |
| InfluxDB | 259.73335 ms | 2.96045 ms | 256.7729 ms | 262.6938 ms | 256.7729 ms | 262.6938 ms | 262.6938 ms | 11,550.3 |
| VM | 36.3654 ms | 2.4076 ms | 33.9578 ms | 38.773 ms | 33.9578 ms | 38.773 ms | 38.773 ms | 82,496.0 |

### Batch/Bulk Writes

![charts/batch_comparison.png](charts/batch_comparison.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 2.56965 ms | 1.02965 ms | 1.54 ms | 3.5993 ms | 1.54 ms | 3.5993 ms | 3.5993 ms | 1,167,474.2 |
| InfluxDB | 10.80395 ms | 624 us | 10.1796 ms | 11.4283 ms | 10.1796 ms | 11.4283 ms | 11.4283 ms | 277,676.2 |
| VM | 278 us | 278 us | 0 s | 556 us | 0 s | 556 us | 556 us | 10,781,671.2 |

### Multi-Sensor Concurrent Writes

![charts/multi_write.png](charts/multi_write.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 2.84295 ms | 780 us | 2.0634 ms | 3.6225 ms | 2.0634 ms | 3.6225 ms | 3.6225 ms | 1,055,241.9 |
| InfluxDB | 7.95075 ms | 223 us | 7.7282 ms | 8.1733 ms | 7.7282 ms | 8.1733 ms | 8.1733 ms | 377,322.9 |
| VM | 514 us | 4 us | 510 us | 517 us | 510 us | 517 us | 517 us | 5,838,847.8 |

## Read Benchmarks

![charts/read_comparison.png](charts/read_comparison.png)

### Single Read (Last 1 Point)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 1.04145 ms | 12 us | 1.0292 ms | 1.0537 ms | 1.0292 ms | 1.0537 ms | 1.0537 ms | 960.2 |
| InfluxDB | 8.72975 ms | 496 us | 8.2333 ms | 9.2262 ms | 8.2333 ms | 9.2262 ms | 9.2262 ms | 114.6 |
| VM | 257 us | 257 us | 0 s | 515 us | 0 s | 515 us | 515 us | 3,884.2 |

### Read-Many (5000 Reads)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 6.979 ms | 1.2161 ms | 5.7629 ms | 8.1951 ms | 5.7629 ms | 8.1951 ms | 8.1951 ms | 3,582,175.1 |
| InfluxDB | 10.62795 ms | 695 us | 9.9325 ms | 11.3234 ms | 9.9325 ms | 11.3234 ms | 11.3234 ms | 2,352,288.1 |
| VM | 258 us | 258 us | 0 s | 516 us | 0 s | 516 us | 516 us | 96,918,007.4 |

## Pub/Sub Benchmark

![charts/pubsub.png](charts/pubsub.png)

| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |
|--------|------|--------|-----|-----|-----|-----|-----|---------|
| GTSDB | 108.1744 ms | 4.2322 ms | 103.9422 ms | 112.4066 ms | 103.9422 ms | 112.4066 ms | 112.4066 ms | 9.2 |
| NSQ | 88.3413 ms | 46 us | 88.2949 ms | 88.3877 ms | 88.2949 ms | 88.3877 ms | 88.3877 ms | 11.3 |

## Key Findings

### Sequential Write Throughput

- #1 **GTSDB**: **35,865 ops/sec** (83.6477 ms)
- #2 **VM**: **19,080 ops/sec** (157.234 ms)
- #3 **InfluxDB**: **2,063 ops/sec** (1.45 s)

### Batch Write Throughput

- #1 **VM**: **10,781,671 ops/sec** (278 us)
- #2 **GTSDB**: **1,167,474 ops/sec** (2.56965 ms)
- #3 **InfluxDB**: **277,676 ops/sec** (10.80395 ms)

### Multi-Sensor Write Throughput

- #1 **VM**: **5,838,848 ops/sec** (514 us)
- #2 **GTSDB**: **1,055,242 ops/sec** (2.84295 ms)
- #3 **InfluxDB**: **377,323 ops/sec** (7.95075 ms)

### Read Performance

- #1 **VM**: **3,884 ops/sec** (257 us)
- #2 **GTSDB**: **960 ops/sec** (1.04145 ms)
- #3 **InfluxDB**: **115 ops/sec** (8.72975 ms)

### Pub/Sub Latency

- #1 **NSQ**: **88.3413 ms** delivery latency
- #2 **GTSDB**: **108.1744 ms** delivery latency

## Head-to-Head Comparison

| Benchmark | Comparison | Ratio | Winner |
|-----------|------------|-------|--------|
| Write (seq) | GTSDB vs InfluxDB | 17.38x | **GTSDB** |
| Write (seq) | GTSDB vs VM | 1.88x | **GTSDB** |
| Pipeline Write | GTSDB vs InfluxDB | 9.31x | **GTSDB** |
| Pipeline Write | GTSDB vs VM | 1.30x | **GTSDB** |
| Batch Write | GTSDB vs InfluxDB | 4.20x | **GTSDB** |
| Batch Write | GTSDB vs VM | 9.24x | **VM** |
| Read (single) | GTSDB vs InfluxDB | 8.38x | **GTSDB** |
| Read (single) | GTSDB vs VM | 4.05x | **VM** |
| Multi-Key Write | GTSDB vs InfluxDB | 2.80x | **GTSDB** |
| Multi-Key Write | GTSDB vs VM | 5.53x | **VM** |
| Pub/Sub | GTSDB vs NSQ | 0.82x latency | **NSQ** |
| Multi-Key Read | GTSDB vs InfluxDB | 1.52x | **GTSDB** |
| Multi-Key Read | GTSDB vs VM | 27.06x | **VM** |
| Write (seq) | InfluxDB vs VM | 9.25x | **VM** |
| Pipeline Write | InfluxDB vs VM | 7.14x | **VM** |
| Batch Write | InfluxDB vs VM | 38.83x | **VM** |
| Read (single) | InfluxDB vs VM | 33.91x | **VM** |
| Multi-Key Write | InfluxDB vs VM | 15.47x | **VM** |
| Multi-Key Read | InfluxDB vs VM | 41.20x | **VM** |