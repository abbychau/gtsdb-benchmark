package main

import (
	"context"
	"time"
)

type Driver interface {
	Name() string
	Connect(ctx context.Context) error
	Close() error
}

type Writer interface {
	Driver
	Write(ctx context.Context, key string, value float64) error
	WriteBatch(ctx context.Context, points []KeyedPoint) error
}

type Reader interface {
	Driver
	Read(ctx context.Context, key string, lastX int) (int, error)
}

type MultiReader interface {
	Driver
	MultiRead(ctx context.Context, keys []string, lastX int) (map[string]int, error)
}

type PubSuber interface {
	Driver
	PubSub(ctx context.Context, key string, count int) (time.Duration, error)
}

type KeyedPoint struct {
	Key       string
	Value     float64
	Timestamp int64
}
