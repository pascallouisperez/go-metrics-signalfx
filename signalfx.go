package signalfx

import (
	"context"
	"fmt"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
)

// Options controls various behavior of the SignalFX bridge.
type Options struct {
	// Duration controls the frequency at which to flush metrics to SignalFX. By
	// defaul, this is set to every 15 seconds.
	Duration time.Duration

	// Logger specifies a logger to use. It is used in verbose mode, and to
	// report flushing errors communicating to SignalFX.
	Logger metrics.Logger

	// Verbose controls the level of verbosity of the publisher. Turning on this
	// option is only recommended for debugging, and should be avoided in production.
	Verbose bool
}

// PublishToSignalFx publishes periodically all the metrics of the specified
// registry to SignalFX (https://signalfx.com/). This is designed to be called
// as a goroutine. Providing a logger is optional and only used to log
// publishing errors.
func PublishToSignalFx(r metrics.Registry, authToken string, options ...Options) {
	var opt Options
	if l := len(options); l > 1 {
		panic("PublishToSignalFx: more than one options provided.")
	} else if l == 1 {
		opt = options[0]
		if opt.Duration == 0 {
			opt.Duration = 15 * time.Second
		}
	}

	publisher := newPublisher(authToken, opt)
	for _ = range time.Tick(opt.Duration) {
		if err := publisher.single(r); err != nil {
			publisher.client = nil
			if opt.Logger != nil {
				opt.Logger.Printf("Unable to publish to SignalFX: %s.", err)
			}
		}
	}
}

type publisher struct {
	authToken string
	client    *sfxclient.HTTPSink
	opt       Options

	// Caches keeping last values sent up to SignalFX.
	// TODO(pascal): use LRU cache, with fixed size.
	last struct {
		counters map[string]int64
		gauges   map[string]int64
		gauges_f map[string]float64
	}
}

func newPublisher(authToken string, opt Options) *publisher {
	p := publisher{authToken: authToken, opt: opt}
	p.last.counters = make(map[string]int64, 0)
	p.last.gauges = make(map[string]int64, 0)
	p.last.gauges_f = make(map[string]float64, 0)
	return &p
}

func (p *publisher) single(r metrics.Registry) error {
	if p.client == nil {
		p.client = sfxclient.NewHTTPSink()
		p.client.AuthToken = p.authToken
	}

	u := p.prepareUpdate()
	r.Each(func(name string, i interface{}) {
		u.metricToDatapoints(name, i)
	})
	return u.flush()
}

type update struct {
	p       *publisher
	ds      []*datapoint.Datapoint
	changes struct {
		counters map[string]int64
		gauges   map[string]int64
		gauges_f map[string]float64
	}
}

func (p *publisher) prepareUpdate() *update {
	u := update{p: p}
	u.changes.counters = make(map[string]int64, 0)
	u.changes.gauges = make(map[string]int64, 0)
	u.changes.gauges_f = make(map[string]float64, 0)
	return &u
}

func (u *update) flush() error {
	// Verbose: log changes.
	if u.p.opt.Verbose && u.p.opt.Logger != nil {
		u.p.opt.Logger.Printf("changes to flush counter=%v, gauges=%v, gauges_f=%v",
			u.changes.counters, u.changes.gauges, u.changes.gauges_f)
	}

	// Publish to SignalFx.
	ctx := context.Background()
	err := u.p.client.AddDatapoints(ctx, u.ds)

	// On error, we flush last values cache to be on the safe side.
	if err != nil {
		for name := range u.changes.counters {
			delete(u.p.last.counters, name)
		}
		for name := range u.changes.gauges {
			delete(u.p.last.gauges, name)
		}
		for name := range u.changes.gauges_f {
			delete(u.p.last.gauges_f, name)
		}
		return err
	}

	// On success, update last values cache.
	for name, counter := range u.changes.counters {
		u.p.last.counters[name] = counter
	}
	for name, gauge := range u.changes.gauges {
		u.p.last.gauges[name] = gauge
	}
	for name, gaugeF := range u.changes.gauges_f {
		u.p.last.gauges_f[name] = gaugeF
	}

	return nil
}

func (u *update) metricToDatapoints(name string, i interface{}) {
	switch metric := i.(type) {
	case metrics.Counter:
		u.appendIfCounterChanged(name, metric.Count())

	case metrics.Gauge:
		u.appendIfGaugeChanged(name, metric.Value())

	case metrics.GaugeFloat64:
		u.appendIfGaugeFChanged(name, metric.Value())

	case metrics.Histogram:
		h := metric.Snapshot()
		ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		u.appendIfCounterChanged(name+".count", h.Count())
		u.appendIfCounterChanged(name+".min", h.Min())
		u.appendIfCounterChanged(name+".max", h.Max())
		u.appendIfGaugeFChanged(name+".mean", h.Mean())
		u.appendIfGaugeFChanged(name+".std-dev", h.StdDev())
		u.appendIfGaugeFChanged(name+".50-percentile", ps[0])
		u.appendIfGaugeFChanged(name+".75-percentile", ps[1])
		u.appendIfGaugeFChanged(name+".95-percentile", ps[2])
		u.appendIfGaugeFChanged(name+".99-percentile", ps[3])
		u.appendIfGaugeFChanged(name+".999-percentile", ps[4])

	case metrics.Meter:
		m := metric.Snapshot()
		u.appendIfCounterChanged(name+".count", m.Count())
		u.appendIfGaugeFChanged(name+".one-minute", m.Rate1())
		u.appendIfGaugeFChanged(name+".five-minute", m.Rate5())
		u.appendIfGaugeFChanged(name+".fifteen-minute", m.Rate15())
		u.appendIfGaugeFChanged(name+".mean-rate", m.RateMean())

	case metrics.Timer:
		t := metric.Snapshot()
		ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		u.appendIfCounterChanged(name+".count", t.Count())
		u.appendIfCounterChanged(name+".min", t.Min())
		u.appendIfCounterChanged(name+".max", t.Max())
		u.appendIfGaugeFChanged(name+".mean", t.Mean())
		u.appendIfGaugeFChanged(name+".std-dev", t.StdDev())
		u.appendIfGaugeFChanged(name+".50-percentile", ps[0])
		u.appendIfGaugeFChanged(name+".75-percentile", ps[1])
		u.appendIfGaugeFChanged(name+".95-percentile", ps[2])
		u.appendIfGaugeFChanged(name+".99-percentile", ps[3])
		u.appendIfGaugeFChanged(name+".999-percentile", ps[4])
		u.appendIfGaugeFChanged(name+".one-minute", t.Rate1())
		u.appendIfGaugeFChanged(name+".five-minute", t.Rate5())
		u.appendIfGaugeFChanged(name+".fifteen-minute", t.Rate15())
		u.appendIfGaugeFChanged(name+".mean-rate", t.RateMean())

	default:
		panic(fmt.Sprintf("Unrecognized metric: %t.", i))
	}
}

func (u *update) appendIfCounterChanged(name string, counter int64) {
	if last, ok := u.p.last.counters[name]; !ok || counter != last {
		u.ds = append(u.ds, sfxclient.Counter(name, nil, counter))
		u.changes.counters[name] = counter
	}
}

func (u *update) appendIfGaugeChanged(name string, gauge int64) {
	if last, ok := u.p.last.gauges[name]; !ok || gauge != last {
		u.ds = append(u.ds, sfxclient.Gauge(name, nil, gauge))
		u.changes.gauges[name] = gauge
	}
}

func (u *update) appendIfGaugeFChanged(name string, gaugeF float64) {
	if last, ok := u.p.last.gauges_f[name]; !ok || gaugeF != last {
		u.ds = append(u.ds, sfxclient.GaugeF(name, nil, gaugeF))
		u.changes.gauges_f[name] = gaugeF
	}
}
