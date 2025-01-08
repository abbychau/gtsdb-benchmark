# GTSDB vs InfluxDB Benchmark

A Go benchmarking tool that compares performance between GTSDB and InfluxDB for time series data operations.

## Features

- Single point write benchmarking
- Multi-sensor parallel write benchmarking 
- Read operation benchmarking
- Comparison of success rates and operation durations

## Requirements

- Go 1.21+
- InfluxDB server running on localhost:8086
- GTSDB server running on localhost:5555

## Dependencies

- github.com/influxdata/influxdb-client-go/v2
- github.com/VictoriaMetrics/metrics (commented out in code)

## Usage

1. Start your InfluxDB and GTSDB servers
2. Configure connection settings in main.go:
   ```go
   gtsdbAddress := "localhost:5555"
   influxURL := "http://localhost:8086"
   influxToken := "your-token-here"
   ```
3. Run the benchmark:
   ```bash
   go run .
   ```

The program will output performance metrics comparing write and read operations between GTSDB and InfluxDB.

## API Examples

See [api.http](api.http) for a complete list of supported GTSDB API operations and example requests.

## Contributing

Pull requests are welcome. I want to add VictoriaMetrics support and more benchmarking features.