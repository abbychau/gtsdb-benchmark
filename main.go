package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type BenchmarkResult struct {
	OperationCount int
	SuccessCount   uint64
	FailureCount   uint64
	Duration       time.Duration
	SuccessRate    float64
}

func connectTCP(address string) (net.Conn, error) {
	return net.Dial("tcp", address)
}

func writeToTCP(conn net.Conn, data []byte) error {
	_, err := conn.Write(append(data, '\n'))
	return err
}

func readFromTCP(conn net.Conn) ([]byte, error) {
	reader := bufio.NewReader(conn)
	return reader.ReadBytes('\n')
}

func main() {

	gtsdbAddress := "localhost:5555"
	influxURL := "http://localhost:8086"
	influxToken := "0sOuyNL1DM4fxLpYlRxvxaKOyjCs3M1VuUICdx81YaNwQLzsrmAKYoJLqUiKGICV4x3SaQFOq12eclj4IB64Qg=="
	nsqAddress := "localhost:4150"

	args := os.Args
	if len(args) > 1 && args[1] == "pubsub" {
		fmt.Println("Running PubSub benchmarks")

		count := 1000000

		result := benchmarkGTSDBPubSub(gtsdbAddress, count)
		fmt.Printf("GTSDB PubSub Performance: %v\n", result.Duration)

		result2 := benchmarkNSQPubSub(nsqAddress, count)
		fmt.Printf("NSQ PubSub Performance: %v\n", result2.Duration)
		return
	}

	if len(args) > 1 && args[1] == "readmany" {
		runReadManyBenchmark(gtsdbAddress, influxURL, influxToken)
		return
	}

	influxClient := influxdb2.NewClient(influxURL, influxToken)

	defer influxClient.Close()

	numPoints := 10000
	sensorID := "benchmark_sensor"

	// Run benchmarks
	gtsdbWriteResult := benchmarkGTSDBWrite(gtsdbAddress, sensorID, numPoints)
	influxWriteResult := benchmarkInfluxWrite(influxClient, sensorID, numPoints)

	gtsdbReadResult := benchmarkGTSDBRead(gtsdbAddress, sensorID)
	influxReadResult := benchmarkInfluxRead(influxClient, sensorID)

	// Print results
	fmt.Printf("Write Performance:\n")
	fmt.Printf("GTSDB: %v (Success Rate: %.2f%%)\n", gtsdbWriteResult.Duration, gtsdbWriteResult.SuccessRate)
	fmt.Printf("InfluxDB: %v (Success Rate: %.2f%%)\n", influxWriteResult.Duration, influxWriteResult.SuccessRate)

	fmt.Printf("\nRead Performance:\n")
	fmt.Printf("GTSDB: %v (Success Rate: %.2f%%)\n", gtsdbReadResult.Duration, gtsdbReadResult.SuccessRate)
	fmt.Printf("InfluxDB: %v (Success Rate: %.2f%%)\n", influxReadResult.Duration, influxReadResult.SuccessRate)

	// Multi-write benchmark
	numPointsPerSensor := 1000
	numSensors := 10

	gtsdbMultiWriteResult := benchmarkGTSDBMultiWrite(gtsdbAddress, numPointsPerSensor, numSensors)
	influxMultiWriteResult := benchmarkInfluxMultiWrite(influxClient, numPointsPerSensor, numSensors)

	fmt.Printf("\nMulti-Write Performance:\n")
	fmt.Printf("GTSDB: %v (Success Rate: %.2f%%)\n", gtsdbMultiWriteResult.Duration, gtsdbMultiWriteResult.SuccessRate)
	fmt.Printf("InfluxDB: %v (Success Rate: %.2f%%)\n", influxMultiWriteResult.Duration, influxMultiWriteResult.SuccessRate)

	// vmSet := metrics.NewSet()
	// vmReadResult := benchmarkVMRead(vmSet, sensorID)
	// vmWriteResult := benchmarkVMWrite(vmSet, sensorID, numPoints)
	// vmMultiWriteResult := benchmarkVMMultiWrite(vmSet, numPointsPerSensor, numSensors)
	// fmt.Printf("Write Performance: VictoriaMetrics: %v (Success Rate: %.2f%%)\n", vmWriteResult.Duration, vmWriteResult.SuccessRate)
	// fmt.Printf("Read Performance: VictoriaMetrics: %v (Success Rate: %.2f%%)\n", vmReadResult.Duration, vmReadResult.SuccessRate)
	// fmt.Printf("Multi-Write Performance: VictoriaMetrics: %v (Success Rate: %.2f%%)\n", vmMultiWriteResult.Duration, vmMultiWriteResult.SuccessRate)
}

// runReadManyBenchmark does 10,000 individual reads (10 sensors x 1,000 each)
// GTSDB via TCP reuse, InfluxDB via HTTP keep-alive
func runReadManyBenchmark(gtsdbAddress, influxURL, influxToken string) {
	const numSensors = 10
	const pointsPerSensor = 1000
	const org = "bench"
	const bucket = "bench"

	fmt.Println("=== Read-Many Benchmark (10,000 individual reads) ===")
	fmt.Println("GTSDB: TCP reuse, lastx=1 per read")
	fmt.Println("InfluxDB: HTTP keep-alive, Flux last() per query")
	fmt.Println()

	// Pre-load data
	fmt.Println("Pre-loading data...")
	InitKeysGTSDB(gtsdbAddress, numSensors)
	time.Sleep(100 * time.Millisecond)
	PreloadGTSDB("localhost:5556", numSensors, pointsPerSensor)
	PreloadInfluxDB(influxURL, influxToken, org, bucket, numSensors, pointsPerSensor)
	fmt.Println("Pre-load done.")

	// GTSDB ReadMany
	fmt.Println("\nGTSDB ReadMany...")
	gtsdbResult := benchmarkGTSDBReadMany(gtsdbAddress, numSensors, pointsPerSensor)

	// InfluxDB ReadMany
	fmt.Println("InfluxDB ReadMany...")
	influxResult := benchmarkInfluxReadMany(influxURL, influxToken, org, bucket, numSensors, pointsPerSensor)

	fmt.Println("\n=== RESULTS ===")
	fmt.Printf("GTSDB:    %v (%d ops, %.2f%% success)\n", gtsdbResult.Duration, gtsdbResult.SuccessCount, gtsdbResult.SuccessRate)
	fmt.Printf("InfluxDB: %v (%d ops, %.2f%% success)\n", influxResult.Duration, influxResult.SuccessCount, influxResult.SuccessRate)
	fmt.Printf("\nGTSDB is %.1fx faster on 10,000 reads\n", float64(influxResult.Duration)/float64(gtsdbResult.Duration))

	fmt.Println("\n=== PAGE DATA ===")
	fmt.Printf("readManyData: GTSDB %.2f / InfluxDB %.2f\n",
		float64(gtsdbResult.Duration.Microseconds())/1000.0,
		float64(influxResult.Duration.Microseconds())/1000.0)
}
