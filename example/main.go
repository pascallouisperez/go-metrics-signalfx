package main

import (
	"fmt"
	"math/rand"
	"time"

	signalfx "github.com/loyyal/go-metrics-signalfx"
	metrics "github.com/rcrowley/go-metrics"
)

type logger struct{}

func (_ logger) Printf(format string, v ...interface{}) {
	fmt.Printf("signalfx: "+format+"\n", v...)
}

func main() {
	go signalfx.PublishToSignalFx(metrics.DefaultRegistry, "<auth_token>", signalfx.Options{
		Duration: 5 * time.Second,
		Logger:   logger{},
		Verbose:  true,
	})

	gauge := metrics.NewGauge()
	metrics.DefaultRegistry.Register("some_metric", gauge)
	for range time.Tick(3 * time.Second) {
		next := rand.Int63() % 3
		fmt.Printf("update some_metric: %d\n", next)
		gauge.Update(next)
	}
}
