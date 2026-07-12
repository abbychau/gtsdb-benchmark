package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	GTSDBAddr    string
	GTSDBHTTP    string
	InfluxURL    string
	InfluxToken  string
	InfluxOrg    string
	InfluxBucket string
	NSQAddr      string
	VMURL        string

	Count   int
	Sensors int
	Runs    int
	Warmup  int

	Format     string
	Databases  []string
	Benchmarks []string
}

var validDBs = map[string]bool{"gtsdb": true, "influx": true, "nsq": true, "vm": true}
var validBenches = map[string]bool{
	"Write (seq)":     true,
	"Read (single)":   true,
	"Batch Write":     true,
	"Multi-Key Write": true,
	"Pipeline Write":  true,
	"Multi-Key Read":  true,
	"Pub/Sub":         true,
	"all":             true,
}

func ParseConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.GTSDBAddr, "gtsdb-addr", "localhost:5555", "GTSDB TCP address")
	flag.StringVar(&cfg.GTSDBHTTP, "gtsdb-http", "localhost:5556", "GTSDB HTTP address")
	flag.StringVar(&cfg.InfluxURL, "influx-url", "http://localhost:8086", "InfluxDB URL")
	flag.StringVar(&cfg.InfluxToken, "influx-token", os.Getenv("INFLUX_TOKEN"), "InfluxDB token (env: INFLUX_TOKEN)")
	flag.StringVar(&cfg.InfluxOrg, "influx-org", "bench", "InfluxDB organization")
	flag.StringVar(&cfg.InfluxBucket, "influx-bucket", "bench", "InfluxDB bucket")
	flag.StringVar(&cfg.NSQAddr, "nsq-addr", "localhost:4150", "NSQ TCP address")
	flag.StringVar(&cfg.VMURL, "vm-url", "http://localhost:8428", "VictoriaMetrics URL")

	flag.IntVar(&cfg.Count, "count", 10000, "Points per write benchmark")
	flag.IntVar(&cfg.Sensors, "sensors", 10, "Number of sensors for multi-write")
	flag.IntVar(&cfg.Runs, "runs", 3, "Number of benchmark runs")
	flag.IntVar(&cfg.Warmup, "warmup", 1000, "Warmup iterations")

	dbStr := flag.String("db", "gtsdb,influx", "Databases: gtsdb,influx,nsq,vm")
	formatStr := flag.String("format", "text", "Output format: text, json")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: benchmark [flags] [benchmark...]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nBenchmarks: Write (seq), Read (single), Batch Write, Multi-Key Write, Pipeline Write, Multi-Key Read, Pub/Sub, all\n")
		fmt.Fprintf(os.Stderr, "\nExample: benchmark -count=100000 -runs=5 \"Write (seq)\" \"Multi-Key Write\"\n")
	}

	flag.Parse()

	cfg.Databases = parseCSV(*dbStr)
	cfg.Format = *formatStr

	cfg.Benchmarks = flag.Args()
	if len(cfg.Benchmarks) == 0 {
		cfg.Benchmarks = []string{"all"}
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func (c *Config) Validate() error {
	for _, db := range c.Databases {
		if !validDBs[db] {
			return fmt.Errorf("unknown database: %s", db)
		}
	}
	for _, b := range c.Benchmarks {
		if !validBenches[b] {
			return fmt.Errorf("unknown benchmark: %s", b)
		}
	}
	if c.Count <= 0 {
		return fmt.Errorf("count must be positive")
	}
	if c.Runs <= 0 {
		return fmt.Errorf("runs must be positive")
	}
	if c.InfluxToken == "" {
		return fmt.Errorf("influx-token is required (set via --influx-token or INFLUX_TOKEN env)")
	}
	return nil
}

func (c *Config) HasDB(name string) bool {
	for _, db := range c.Databases {
		if db == name {
			return true
		}
	}
	return false
}

func (c *Config) HasBench(name string) bool {
	if contains(c.Benchmarks, "all") {
		return true
	}
	return contains(c.Benchmarks, name)
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
