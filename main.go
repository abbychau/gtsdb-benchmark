package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	cfg := ParseConfig()
	if cfg.InfluxToken == "" {
		fmt.Fprintln(os.Stderr, "Warning: INFLUX_TOKEN not set. Skipping InfluxDB benchmarks.")
	}

	var results []*BenchmarkResult

	if cfg.HasDB("gtsdb") {
		g := newGTSDBDriver(cfg.GTSDBAddr)
		if err := g.Connect(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "GTSDB: %v\n", err)
		} else {
			defer g.Close()
			runGTSDBBenchmarks(cfg, g, &results)
		}
	}

	if cfg.HasDB("influx") {
		i := newInfluxDriver(cfg.InfluxURL, cfg.InfluxToken, cfg.InfluxOrg, cfg.InfluxBucket)
		if err := i.Connect(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "InfluxDB: %v\n", err)
		} else {
			defer i.Close()
			runInfluxBenchmarks(cfg, i, &results)
		}
	}

	if cfg.HasDB("nsq") {
		n := newNSQDriver(cfg.NSQAddr)
		if cfg.HasBench("Pub/Sub") {
			r := runPubSubBenchmark(n, benchSensorKey, cfg.Count, cfg.Runs)
			results = append(results, r)
		}
	}

	if cfg.HasDB("vm") {
		v := newVMDriver(cfg.VMURL)
		if err := v.Connect(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "VictoriaMetrics: %v\n", err)
		} else {
			defer v.Close()
			runVMBenchmarks(cfg, v, &results)
		}
	}

	printReport(cfg.Format, results)
	printComparison(results)
}

func runGTSDBBenchmarks(cfg *Config, g *gtsdbDriver, results *[]*BenchmarkResult) {
	if cfg.HasBench("Write (seq)") {
		r := runWriteBenchmark(g, benchSensorKey, cfg.Count, cfg.Warmup, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Pipeline Write") {
		r := runPipelinedWriteGTSDB(cfg.GTSDBAddr, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Batch Write") {
		r := runBatchWrite(g, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Read (single)") {
		r := runReadBenchmark(g, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Multi-Key Write") {
		r := runMultiWriteGTSDBBatch(g, cfg.Count/cfg.Sensors, cfg.Sensors, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Pub/Sub") {
		r := runPubSubBenchmark(g, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Multi-Key Read") {
		preloadAndInit(cfg, g, nil, nil)
		r := runMultiReadGTSDB(g, cfg.Sensors, 5000, cfg.Runs)
		*results = append(*results, r)
	}
}

func runInfluxBenchmarks(cfg *Config, i *influxDriver, results *[]*BenchmarkResult) {
	if cfg.HasBench("Write (seq)") {
		r := runWriteBenchmark(i, benchSensorKey, cfg.Count, cfg.Warmup, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Pipeline Write") {
		r := runPipelinedWrite(i, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Batch Write") {
		r := runBatchWrite(i, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Read (single)") {
		r := runReadBenchmark(i, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Multi-Key Write") {
		r := runMultiWriteInflux(i, cfg.Count/cfg.Sensors, cfg.Sensors, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Multi-Key Read") {
		preloadAndInit(cfg, nil, i, nil)
		r := runReadManyInflux(i, cfg.Sensors, 5000, cfg.Runs)
		*results = append(*results, r)
	}
}

func runVMBenchmarks(cfg *Config, v *vmDriver, results *[]*BenchmarkResult) {
	if cfg.HasBench("Write (seq)") {
		r := runWriteBenchmark(v, benchSensorKey, cfg.Count, cfg.Warmup, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Pipeline Write") {
		r := runPipelinedWrite(v, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Batch Write") {
		r := runBatchWrite(v, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Read (single)") {
		r := runReadBenchmark(v, benchSensorKey, cfg.Count, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Multi-Key Write") {
		r := runMultiWriteVM(v, cfg.Count/cfg.Sensors, cfg.Sensors, cfg.Runs)
		*results = append(*results, r)
	}

	if cfg.HasBench("Multi-Key Read") {
		preloadAndInit(cfg, nil, nil, v)
		r := runReadManyVM(v, cfg.Sensors, 5000, cfg.Runs)
		*results = append(*results, r)
	}
}
