package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	json "github.com/bytedance/sonic"
)

type reportEntry struct {
	Name        string  `json:"name"`
	Driver      string  `json:"driver"`
	Runs        int     `json:"runs"`
	Ops         int     `json:"ops_per_run"`
	Min         string  `json:"min"`
	Max         string  `json:"max"`
	Mean        string  `json:"mean"`
	StdDev      string  `json:"stddev"`
	P50         string  `json:"p50"`
	P95         string  `json:"p95"`
	P99         string  `json:"p99"`
	OpsPerSec   float64 `json:"ops_per_sec"`
	SuccessRate float64 `json:"success_rate"`
}

func newReportEntry(r *BenchmarkResult) reportEntry {
	return reportEntry{
		Name:        r.Name,
		Driver:      r.DriverName,
		Runs:        len(r.Durations),
		Ops:         r.OperationCount,
		Min:         r.Min.String(),
		Max:         r.Max.String(),
		Mean:        r.Mean.String(),
		StdDev:      r.StdDev.String(),
		P50:         r.P50.String(),
		P95:         r.P95.String(),
		P99:         r.P99.String(),
		OpsPerSec:   r.OpsPerSec,
		SuccessRate: r.SuccessRate(),
	}
}

func printReport(format string, results []*BenchmarkResult) {
	switch format {
	case "json":
		printJSON(results)
	default:
		printTable(results)
	}
}

func printTable(results []*BenchmarkResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Benchmark\tDriver\tRuns\tOps/Run\tMean\tStdDev\tP50\tP95\tP99\tOps/sec\tSuccess%%\n")
	fmt.Fprintf(w, "---------\t------\t----\t-------\t----\t------\t---\t---\t---\t-------\t--------\n")

	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%.0f\t%.2f%%\n",
			r.Name,
			r.DriverName,
			len(r.Durations),
			r.OperationCount,
			r.Mean,
			r.StdDev,
			r.P50,
			r.P95,
			r.P99,
			r.OpsPerSec,
			r.SuccessRate(),
		)
	}
	w.Flush()
}

func printJSON(results []*BenchmarkResult) {
	entries := make([]reportEntry, len(results))
	for i, r := range results {
		entries[i] = newReportEntry(r)
	}
	enc := json.ConfigDefault.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(entries)
}

func printComparison(results []*BenchmarkResult) {
	groups := make(map[string][]*BenchmarkResult)
	for _, r := range results {
		groups[r.Name] = append(groups[r.Name], r)
	}

	fmt.Println("\n=== COMPARISON ===")
	for benchName, group := range groups {
		if len(group) < 2 {
			continue
		}
		fmt.Printf("\n%s:\n", strings.ToUpper(benchName))
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]
				if b.OpsPerSec > 0 {
					ratio := a.OpsPerSec / b.OpsPerSec
					fmt.Printf("  %s vs %s: %.2fx (%s: %.0f ops/s, %s: %.0f ops/s)\n",
						a.DriverName, b.DriverName, ratio,
						a.DriverName, a.OpsPerSec,
						b.DriverName, b.OpsPerSec,
					)
				}
			}
		}
	}
}
