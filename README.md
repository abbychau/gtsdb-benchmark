# GTSDB Benchmark

A Go benchmarking tool that compares performance between GTSDB, InfluxDB, NSQ, and VictoriaMetrics for time series data operations.

## Features

- Single-point write benchmarking
- Pipelined write benchmarking (GTSDB)
- Batch/bulk write benchmarking
- Multi-sensor concurrent write benchmarking
- Read operation benchmarking (single + read-many)
- PubSub latency comparison (GTSDB vs NSQ)
- Multi-run statistical analysis (min, max, mean, stddev, p50, p95, p99)
- Text table and JSON output formats

## Requirements

- Go 1.22+
- Target servers running (as needed):
  - GTSDB on `localhost:5555` (TCP) and `localhost:5556` (HTTP)
  - InfluxDB on `localhost:8086`
  - NSQ on `localhost:4150`
  - VictoriaMetrics on `localhost:8428`

## Usage

### Quick Start

```bash
# Run all benchmarks
go run . all

# Run specific benchmarks
go run . write batch multi pubsub

# Custom parameters
go run . -count=100000 -runs=5 write pipeline batch

# JSON output
go run . -format=json all

# Target specific databases
go run . -db=gtsdb,influx write multi

# Help
go run . -help
```

### Configuration

| Flag            | Default                      | Description                          |
|----------------|------------------------------|--------------------------------------|
| `--gtsdb-addr`  | `localhost:5555`             | GTSDB TCP address                    |
| `--gtsdb-http`  | `localhost:5556`             | GTSDB HTTP address (batch writes)    |
| `--influx-url`  | `http://localhost:8086`      | InfluxDB URL                         |
| `--influx-token`| `$INFLUX_TOKEN`              | InfluxDB token (required)            |
| `--influx-org`  | `bench`                      | InfluxDB organization                |
| `--influx-bucket`| `bench`                     | InfluxDB bucket                      |
| `--nsq-addr`    | `localhost:4150`             | NSQ TCP address                      |
| `--vm-url`      | `http://localhost:8428`      | VictoriaMetrics URL                  |
| `--db`          | `gtsdb,influx`               | Databases to benchmark               |
| `--count`       | `10000`                      | Points per write benchmark           |
| `--sensors`     | `10`                         | Sensors for multi-write              |
| `--runs`        | `3`                          | Number of benchmark runs             |
| `--warmup`      | `1000`                       | Warmup iterations                    |
| `--format`      | `text`                       | Output format: `text` or `json`      |

### Environment Variables

- `INFLUX_TOKEN` — InfluxDB authentication token (required for Influx benchmarks)

### Benchmarks

| Name       | Description                                      | Applicable DBs      |
|------------|--------------------------------------------------|---------------------|
| `write`    | Sequential single-point writes                   | gtsdb, influx, vm   |
| `pipeline` | Pipelined writes (send all, read all ACKs)       | gtsdb               |
| `batch`    | Batch/bulk writes                                | gtsdb, influx, vm   |
| `read`     | Individual read queries                          | gtsdb, influx, vm   |
| `multi`    | Multi-sensor concurrent writes                   | gtsdb, influx, vm   |
| `readmany` | 10,000 individual `lastx=1` reads                | gtsdb, influx       |
| `pubsub`   | PubSub message delivery latency                  | gtsdb, nsq          |
| `all`      | Run all applicable benchmarks                    | all                 |

### Example Output (text format)

```
Benchmark  Driver    Runs  Ops/Run  Mean       StdDev     P50       P95       P99       Ops/sec    Success%
---------  ------    ----  -------  ----       ------     ---       ---       ---       -------    --------
write      GTSDB     3     10000    1.234s     45ms       1.230s    1.280s    1.290s    8103       100.00%
write      InfluxDB  3     10000    2.567s     89ms       2.560s    2.600s    2.610s    3895       100.00%
pipeline   GTSDB     3     10000    234ms      12ms       231ms     245ms     248ms     42735      100.00%
batch      GTSDB     3     10000    89ms       5ms        88ms      94ms      95ms      112359     100.00%
batch      InfluxDB  3     10000    156ms      8ms        154ms     162ms     164ms     64102      100.00%

=== COMPARISON ===
WRITE:
  GTSDB vs InfluxDB: 2.08x (GTSDB: 8103 ops/s, InfluxDB: 3895 ops/s)
```

### Example Output (JSON format)

```json
[
  {
    "name": "write",
    "driver": "GTSDB",
    "runs": 3,
    "ops_per_run": 10000,
    "min": "1.204s",
    "max": "1.278s",
    "mean": "1.234s",
    "stddev": "32ms",
    "p50": "1.230s",
    "p95": "1.278s",
    "p99": "1.278s",
    "ops_per_sec": 8103.73,
    "success_rate": 100
  }
]
```

## Contributing

Pull requests are welcome for additional database drivers and benchmark types.
