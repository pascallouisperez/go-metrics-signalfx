# SignalFX bridge for go-metrics

[![Build Status](https://travis-ci.org/loyyal/go-metrics-signalfx.svg?branch=master)](https://travis-ci.org/loyyal/go-metrics-signalfx)
[![GoDoc](https://godoc.org/github.com/loyyal/go-metrics-signalfx?status.svg)](https://godoc.org/github.com/loyyal/go-metrics-signalfx)

Simply use as follows

	import (
		"time"

		signalfx "github.com/loyyal/go-metrics-signalfx"
		metrics "github.com/rcrowley/go-metrics"
	)

	...

	go signalfx.PublishToSignalFx(metrics.DefaultRegistry, 15 * time.Second, nil, "<auth_token>")
