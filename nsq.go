package main

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/nsqio/go-nsq"
)

type NSQMessage struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

func benchmarkNSQPubSub(nsqAddress string, count int) BenchmarkResult {
	result := BenchmarkResult{}
	done := make(chan bool)

	producer, err := nsq.NewProducer(nsqAddress, nsq.NewConfig())
	if err != nil {
		return result
	}

	consumer, err := nsq.NewConsumer("test", "benchmark", nsq.NewConfig())
	if err != nil {
		return result
	}

	producer.SetLogger(nil, 0)
	consumer.SetLogger(nil, 0)

	start := time.Now()
	var received atomic.Int64
	received.Store(0)

	consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		newCount := received.Add(1)
		if newCount == int64(count) {
			result.Duration = time.Since(start)
			producer.Stop()
			consumer.Stop()
			done <- true
		}
		return nil
	}))

	if err = consumer.ConnectToNSQD(nsqAddress); err != nil {
		return result
	}

	// Wait for subscription to be established
	time.Sleep(1 * time.Second)

	// Start publisher
	go func() {
		for i := 0; i < count; i++ {
			msg := NSQMessage{
				Key:   "benchmark_sensor",
				Value: float64(i),
			}
			if data, err := json.Marshal(msg); err == nil {
				producer.Publish("test", data)
			}
		}
	}()

	<-done // Wait for benchmark to complete
	return result
}
