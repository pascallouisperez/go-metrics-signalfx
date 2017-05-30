package signalfx

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Zuite struct{}

var _ = Suite(&Zuite{})

func (s *Zuite) TestAppendIfCounterChanged_caching(c *C) {
	p := newPublisher("", Options{})
	var u *update

	// Not in cache.
	u = p.prepareUpdate()
	u.appendIfCounterChanged("not_in_cache", 5)

	c.Assert(u.ds, HasLen, 1)
	c.Assert(u.ds[0].Metric, Equals, "not_in_cache")
	c.Assert(u.ds[0].Value.String(), Equals, "5")

	c.Assert(u.changes.gauges, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 0)
	c.Assert(u.changes.counters, HasLen, 1)
	c.Assert(u.changes.counters["not_in_cache"], Equals, int64(5))

	// In cache, different value.
	p.last.counters["in_cache_diff_value"] = 4

	u = p.prepareUpdate()
	u.appendIfCounterChanged("in_cache_diff_value", 5)

	c.Assert(u.ds, HasLen, 1)
	c.Assert(u.ds[0].Metric, Equals, "in_cache_diff_value")
	c.Assert(u.ds[0].Value.String(), Equals, "5")

	c.Assert(u.changes.gauges, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 0)
	c.Assert(u.changes.counters, HasLen, 1)
	c.Assert(u.changes.counters["in_cache_diff_value"], Equals, int64(5))

	// In cache, same value.
	p.last.counters["in_cache_same_value"] = 5

	u = p.prepareUpdate()
	u.appendIfCounterChanged("in_cache_same_value", 5)

	c.Assert(u.ds, HasLen, 0)

	c.Assert(u.changes.counters, HasLen, 0)
	c.Assert(u.changes.gauges, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 0)
}

func (s *Zuite) TestAppendIfGaugeChanged_caching(c *C) {
	p := newPublisher("", Options{})
	var u *update

	// Not in cache.
	u = p.prepareUpdate()
	u.appendIfGaugeChanged("not_in_cache", 5)

	c.Assert(u.ds, HasLen, 1)
	c.Assert(u.ds[0].Metric, Equals, "not_in_cache")
	c.Assert(u.ds[0].Value.String(), Equals, "5")

	c.Assert(u.changes.counters, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 0)
	c.Assert(u.changes.gauges, HasLen, 1)
	c.Assert(u.changes.gauges["not_in_cache"], Equals, int64(5))

	// In cache, different value.
	p.last.gauges_f["in_cache_diff_value"] = 4

	u = p.prepareUpdate()
	u.appendIfGaugeChanged("in_cache_diff_value", 5)

	c.Assert(u.ds, HasLen, 1)
	c.Assert(u.ds[0].Metric, Equals, "in_cache_diff_value")
	c.Assert(u.ds[0].Value.String(), Equals, "5")

	c.Assert(u.changes.counters, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 0)
	c.Assert(u.changes.gauges, HasLen, 1)
	c.Assert(u.changes.gauges["in_cache_diff_value"], Equals, int64(5))

	// In cache, same value.
	p.last.gauges["in_cache_same_value"] = 5

	u = p.prepareUpdate()
	u.appendIfGaugeChanged("in_cache_same_value", 5)

	c.Assert(u.ds, HasLen, 0)

	c.Assert(u.changes.counters, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 0)
	c.Assert(u.changes.gauges, HasLen, 0)
}

func (s *Zuite) TestAppendIfGaugeFChanged_caching(c *C) {
	p := newPublisher("", Options{})
	var u *update

	// Not in cache.
	u = p.prepareUpdate()
	u.appendIfGaugeFChanged("not_in_cache", 5)

	c.Assert(u.ds, HasLen, 1)
	c.Assert(u.ds[0].Metric, Equals, "not_in_cache")
	c.Assert(u.ds[0].Value.String(), Equals, "5")

	c.Assert(u.changes.counters, HasLen, 0)
	c.Assert(u.changes.gauges, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 1)
	c.Assert(u.changes.gauges_f["not_in_cache"], Equals, float64(5))

	// In cache, different value.
	p.last.gauges_f["in_cache_diff_value"] = 4

	u = p.prepareUpdate()
	u.appendIfGaugeFChanged("in_cache_diff_value", 5)

	c.Assert(u.ds, HasLen, 1)
	c.Assert(u.ds[0].Metric, Equals, "in_cache_diff_value")
	c.Assert(u.ds[0].Value.String(), Equals, "5")

	c.Assert(u.changes.counters, HasLen, 0)
	c.Assert(u.changes.gauges, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 1)
	c.Assert(u.changes.gauges_f["in_cache_diff_value"], Equals, float64(5))

	// In cache, same value.
	p.last.gauges_f["in_cache_same_value"] = 5

	u = p.prepareUpdate()
	u.appendIfGaugeFChanged("in_cache_same_value", 5)

	c.Assert(u.ds, HasLen, 0)

	c.Assert(u.changes.counters, HasLen, 0)
	c.Assert(u.changes.gauges, HasLen, 0)
	c.Assert(u.changes.gauges_f, HasLen, 0)
}
