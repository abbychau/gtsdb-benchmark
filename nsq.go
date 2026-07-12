package main

import (
	"context"
	"sync/atomic"
	"time"

	json "github.com/bytedance/sonic"

	"github.com/nsqio/go-nsq"
)

type nsqDriver struct {
	addr string
}

func newNSQDriver(addr string) *nsqDriver {
	return &nsqDriver{addr: addr}
}

func (d *nsqDriver) Name() string { return "NSQ" }

func (d *nsqDriver) Connect(ctx context.Context) error {
	return nil
}

func (d *nsqDriver) Close() error { return nil }

func (d *nsqDriver) PubSub(ctx context.Context, key string, count int) (time.Duration, error) {
	producer, err := nsq.NewProducer(d.addr, nsq.NewConfig())
	if err != nil {
		return 0, err
	}
	defer producer.Stop()

	consumer, err := nsq.NewConsumer("test", "benchmark", nsq.NewConfig())
	if err != nil {
		return 0, err
	}

	producer.SetLogger(nil, 0)
	consumer.SetLogger(nil, 0)

	var received atomic.Int64
	done := make(chan time.Time, 1)

	consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		n := received.Add(1)
		if n == int64(count) {
			done <- time.Now()
		}
		return nil
	}))

	if err = consumer.ConnectToNSQD(d.addr); err != nil {
		return 0, err
	}
	defer consumer.Stop()

	time.Sleep(100 * time.Millisecond)

	start := time.Now()

	go func() {
		for i := 0; i < count; i++ {
			msg := struct {
				Key   string  `json:"key"`
				Value float64 `json:"value"`
			}{Key: key, Value: float64(i)}
			data, _ := json.Marshal(msg)
			producer.Publish("test", data)
		}
	}()

	select {
	case end := <-done:
		return end.Sub(start), nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
