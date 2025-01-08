package main

import (
	"bufio"
	"fmt"
	"net"
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
	// Initialize clients
	gtsdbAddress := "localhost:5555"
	influxURL := "http://localhost:8086"
	influxToken := "0sOuyNL1DM4fxLpYlRxvxaKOyjCs3M1VuUICdx81YaNwQLzsrmAKYoJLqUiKGICV4x3SaQFOq12eclj4IB64Qg=="
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
