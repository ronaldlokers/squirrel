package squirrel

import "expvar"

var (
	metricSpoolDepth        = expvar.NewInt("spool_depth")
	metricDrainDeferred     = expvar.NewInt("drain_deferred_total")
	metricDrainBackoffMS    = expvar.NewInt("drain_backoff_ms")
	metricPushFailuresTotal = expvar.NewInt("push_failures_total")
)

func RecordPushFailure() {
	metricPushFailuresTotal.Add(1)
}
