package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	json "github.com/bytedance/sonic"
)

type gtsdbDriver struct {
	tcpAddr string
	conn    net.Conn
	reader  *bufio.Reader
	mu      sync.Mutex
}

func newGTSDBDriver(tcpAddr string) *gtsdbDriver {
	return &gtsdbDriver{
		tcpAddr: tcpAddr,
	}
}

func (d *gtsdbDriver) Name() string { return "GTSDB" }

func (d *gtsdbDriver) Connect(ctx context.Context) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", d.tcpAddr)
	if err != nil {
		return fmt.Errorf("gtsdb connect: %w", err)
	}
	d.conn = conn
	d.reader = bufio.NewReader(conn)
	return nil
}

func (d *gtsdbDriver) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

func (d *gtsdbDriver) Write(ctx context.Context, key string, value float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.writeLocked(key, value)
}

func (d *gtsdbDriver) writeLocked(key string, value float64) error {
	payload := fmt.Sprintf(`{"operation":"write","key":"%s","write":{"value":%f}}`, key, value)
	if _, err := d.conn.Write(append([]byte(payload), '\n')); err != nil {
		return err
	}
	_, err := d.reader.ReadBytes('\n')
	return err
}

func (d *gtsdbDriver) writePipelined(ctx context.Context, key string, values []float64) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, v := range values {
		payload := fmt.Sprintf(`{"operation":"write","key":"%s","write":{"value":%f}}`, key, v)
		if _, err := d.conn.Write(append([]byte(payload), '\n')); err != nil {
			return 0, err
		}
	}

	success := 0
	for i := 0; i < len(values); i++ {
		if _, err := d.reader.ReadBytes('\n'); err != nil {
			return success, err
		}
		success++
	}
	return success, nil
}

func (d *gtsdbDriver) WriteBatch(ctx context.Context, points []KeyedPoint) error {
	return d.writeBatchTCP(ctx, points)
}

// writeBatchTCP sends a batch-write via a fresh TCP connection (avoids shared-state issues).
func (d *gtsdbDriver) writeBatchTCP(ctx context.Context, points []KeyedPoint) error {
	return gtsdbBatchWriteFresh(d.tcpAddr, points)
}

// gtsdbBatchWriteFresh opens a new TCP connection, sends batch-write(s), and closes it.
// Sends all points in chunks of up to 10000 (GTSDB's batch limit).
func gtsdbBatchWriteFresh(tcpAddr string, points []KeyedPoint) error {
	const pointsPerChunk = 10000

	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for i := 0; i < len(points); i += pointsPerChunk {
		end := i + pointsPerChunk
		if end > len(points) {
			end = len(points)
		}
		chunk := points[i:end]

		var sb strings.Builder
		sb.WriteString(`{"operation":"batch-write","points":[`)
		for j, p := range chunk {
			if j > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(fmt.Sprintf(`{"key":"%s","value":%f,"timestamp":%d}`, p.Key, p.Value, p.Timestamp))
		}
		sb.WriteString(`]}`)
		payload := sb.String()

		_, err = conn.Write(append([]byte(payload), '\n'))
		if err != nil {
			return err
		}

		resp, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}

		var result struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return fmt.Errorf("batch-write: parse error: %w", err)
		}
		if !result.Success {
			return fmt.Errorf("batch-write failed: %s", result.Message)
		}
	}
	return nil
}

// gtsdbReadResponse matches the JSON structure returned by GTSDB's read operation.
type gtsdbReadResponse struct {
	Success bool             `json:"success"`
	Data    []gtsdbDataPoint `json:"data"`
}

type gtsdbDataPoint struct {
	Key       string  `json:"key"`
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

func (d *gtsdbDriver) Read(ctx context.Context, key string, lastX int) (int, error) {
	// Use shared connection with mutex for thread safety
	d.mu.Lock()
	defer d.mu.Unlock()

	payload := fmt.Sprintf(`{"operation":"read","key":"%s","read":{"lastx":%d}}`, key, lastX)
	if _, err := d.conn.Write(append([]byte(payload), '\n')); err != nil {
		return 0, err
	}
	resp, err := d.reader.ReadBytes('\n')
	if err != nil {
		return 0, err
	}

	var result gtsdbReadResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}
	return len(result.Data), nil
}

// gtsdbReadFresh opens a new TCP connection, performs a read, and closes it.
func gtsdbReadFresh(tcpAddr, key string, lastX int) (int, error) {
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	payload := fmt.Sprintf(`{"operation":"read","key":"%s","read":{"lastx":%d}}`, key, lastX)
	if _, err := conn.Write(append([]byte(payload), '\n')); err != nil {
		return 0, err
	}
	resp, err := reader.ReadBytes('\n')
	if err != nil {
		return 0, err
	}

	var result gtsdbReadResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}
	return len(result.Data), nil
}

// MultiRead uses GTSDB's multi-read API to read from multiple keys in one TCP round-trip.
func (d *gtsdbDriver) MultiRead(ctx context.Context, keys []string, lastX int) (map[string]int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	keysJSON, _ := json.Marshal(keys)
	payload := fmt.Sprintf(`{"operation":"multi-read","keys":%s,"read":{"lastx":%d}}`, string(keysJSON), lastX)
	if _, err := d.conn.Write(append([]byte(payload), '\n')); err != nil {
		return nil, err
	}
	resp, err := d.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var result struct {
		Success   bool                        `json:"success"`
		MultiData map[string][]gtsdbDataPoint `json:"multi_data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(result.MultiData))
	for k, points := range result.MultiData {
		counts[k] = len(points)
	}
	return counts, nil
}

func (d *gtsdbDriver) PubSub(ctx context.Context, key string, count int) (time.Duration, error) {
	pubConn, err := net.Dial("tcp", d.tcpAddr)
	if err != nil {
		return 0, err
	}
	defer pubConn.Close()
	pubReader := bufio.NewReader(pubConn)

	subConn, err := net.Dial("tcp", d.tcpAddr)
	if err != nil {
		return 0, err
	}
	defer subConn.Close()
	subReader := bufio.NewReader(subConn)

	subPayload := fmt.Sprintf(`{"operation":"subscribe","key":"%s"}`, key)
	if _, err := subConn.Write(append([]byte(subPayload), '\n')); err != nil {
		return 0, err
	}
	if _, err := subReader.ReadBytes('\n'); err != nil {
		return 0, err
	}

	time.Sleep(100 * time.Millisecond)

	received := make(chan struct{})
	go func() {
		for i := 0; i < count; i++ {
			subReader.ReadBytes('\n')
		}
		close(received)
	}()

	margin := int(0.1 * float64(count))
	total := count + margin

	start := time.Now()
	for i := 0; i < total; i++ {
		pubPayload := fmt.Sprintf(`{"operation":"write","key":"%s","write":{"value":%f}}`, key, float64(i))
		pubConn.Write(append([]byte(pubPayload), '\n'))
		pubReader.ReadBytes('\n')
	}

	select {
	case <-received:
		return time.Since(start), nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (d *gtsdbDriver) initKeys(numSensors int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := 0; i < numSensors; i++ {
		cmd := fmt.Sprintf(`{"operation":"initkey","key":"bench_sensor_%d"}`, i)
		d.conn.Write(append([]byte(cmd), '\n'))
		d.reader.ReadBytes('\n')
	}
	return nil
}

func (d *gtsdbDriver) preloadTCP(numSensors, pointsPerSensor int) error {
	for i := 0; i < numSensors; i++ {
		var points []KeyedPoint
		for j := 0; j < pointsPerSensor; j++ {
			points = append(points, KeyedPoint{
				Key:       fmt.Sprintf("bench_sensor_%d", i),
				Value:     float64(j) * 1.5,
				Timestamp: 1700000000 + int64(j),
			})
		}
		if err := gtsdbBatchWriteFresh(d.tcpAddr, points); err != nil {
			return err
		}
	}
	return nil
}
