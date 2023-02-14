package selfmon

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"sort"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/sirupsen/logrus"
	"github.com/toni-moreno/oracle_collector/pkg/agent/output"
	"github.com/toni-moreno/oracle_collector/pkg/config"
	"github.com/toni-moreno/oracle_collector/pkg/utils"
)

var (
	log    *logrus.Logger
	conf   *config.SelfMonConfig
	chExit chan bool
	// memory for GVM data colletion
	lastSampleTime time.Time
	lastPauseNs    uint64
	lastNumGc      uint32
)

// SetLogger set log SELF_MON
func SetLogger(l *logrus.Logger) {
	log = l
}

func Init(cfg *config.SelfMonConfig) {
	conf = cfg
	chExit = make(chan bool)
	lastSampleTime = time.Now()
	log.Infof("[SELF_MON] Self Monitor Enabled  %t", conf.Enabled)
	if conf.Enabled {
		go startSelfmonCollector()
	}
}

// End release SELF_MON process
func End() {
	close(chExit)
}

func telegrafFieldsFromRTSamples(samples []metrics.Sample) map[string]interface{} {
	fields := make(map[string]interface{}, len(samples))

	for _, sample := range samples {
		// Pull out the name and value.
		name, value := sample.Name, sample.Value

		field_name := strings.ReplaceAll(filepath.Base(name), ":", "_")
		var field_value interface{}

		// Handle each sample.
		switch value.Kind() {
		case metrics.KindUint64:
			field_value = value.Uint64()
		case metrics.KindFloat64:
			field_value = value.Float64()
		case metrics.KindFloat64Histogram:
			// The histogram may be quite large, so let's just pull out
			// a crude estimate for the median for the sake of this example.
			field_value = medianBucket(value.Float64Histogram())
		case metrics.KindBad:
			// This should never happen because all metrics are supported
			// by construction.
			panic("bug in runtime/metrics package!")
		default:
			// This may happen as new metrics get added.
			//
			// The safest thing to do here is to simply log it somewhere
			// as something to look into, but ignore it for now.
			// In the worst case, you might temporarily miss out on a new metric.
			log.Warnf("%s: unexpected metric Kind: %v\n", name, value.Kind())
		}
		fields[field_name] = field_value
	}
	return fields
}

func collectRuntimeStats() (int, error) {
	descs := metrics.All()

	// Create a sample for each metric.
	samples := make([]metrics.Sample, len(descs))
	for i := range samples {
		samples[i].Name = descs[i].Name
	}

	// Sample the metrics. Re-use the samples slice if you can!
	now := time.Now()
	metrics.Read(samples)

	// Get
	meas_map := make(map[string][]metrics.Sample)

	// Iterate over all results.
	for _, sample := range samples {
		// Pull out the name and value.

		meas_name := "runtime_" + strings.ReplaceAll(utils.TrimLeftChar(filepath.Dir(sample.Name)), "/", "_")
		if len(conf.Prefix) > 0 {
			meas_name = conf.Prefix + meas_name
		}
		meas_map[meas_name] = append(meas_map[meas_name], sample)

	}

	sm := []telegraf.Metric{}

	for meas_name, rt_samples := range meas_map {

		fields := telegrafFieldsFromRTSamples(rt_samples)
		metric := metric.New(meas_name, conf.ExtraLabels, fields, now)

		sm = append(sm, metric)
	}

	output.SendMetrics(sm)
	return 0, nil
}

func medianBucket(h *metrics.Float64Histogram) float64 {
	total := uint64(0)
	for _, count := range h.Counts {
		total += count
	}
	thresh := total / 2
	total = 0
	for i, count := range h.Counts {
		total += count
		if total >= thresh {
			return h.Buckets[i]
		}
	}
	panic("should not happen")
}

func startSelfmonCollector() {
	flushTicker := time.NewTicker(conf.ReqPeriod)
	defer flushTicker.Stop()

	log.Infof("[SELF_MON] beginning Selfmonitor Collector thead")
	for {
		select {
		case <-chExit:
			// need to flush all data
			n, err := collectRuntimeStats()
			if err != nil {
				log.Infof("[SELF_MON] Flushed %d runtime metrics: with error:%s", n, err)
			} else {
				log.Infof("[SELF_MON] Flushed %d runtime metrics: OK", n)
			}
			n, err = collectLegacyRuntimeStats()
			if err != nil {
				log.Infof("[SELF_MON] Flushed %d Legacy runtime metrics: with error:%s", n, err)
			} else {
				log.Infof("[SELF_MON] Flushed %d Legacy runtime metrics: OK", n)
			}

			return
		case <-flushTicker.C:
			/*
				n, err := collectRuntimeStats()
				if err != nil {
					log.Infof("[SELF_MON] Flushed %d metrics: with error:%s", n, err)
				} else {
					log.Infof("[SELF_MON] Flushed %d metrics: OK", n)
				}*/
			n, err := collectLegacyRuntimeStats()
			if err != nil {
				log.Infof("[SELF_MON] Flushed %d Legacy runtime metrics: with error:%s", n, err)
			} else {
				log.Infof("[SELF_MON] Flushed %d Legacy runtime metrics: OK", n)
			}

		}
	}
}

func SendQueryStat(extraLabels map[string]string, mgc *config.OracleMetricGroupConfig, mc *config.OracleMetricConfig, n int, t time.Duration) {
	result := []telegraf.Metric{}

	tags := make(map[string]string)
	// first added General extra tags from
	for k, v := range extraLabels {
		tags[k] = v
	}
	// and then added Extra tags from sefl-monitor config
	for k, v := range conf.ExtraLabels {
		tags[k] = v
	}

	tags["metric_group"] = mgc.Name
	tags["metric_context"] = mc.Context // maintained for compatibility
	tags["metric_id"] = mc.ID
	fields := make(map[string]interface{})
	fields["num_metrics"] = n
	fields["duration_us"] = t.Microseconds()
	now := time.Now()
	meas_name := "collect_stats"
	if len(conf.Prefix) > 0 {
		meas_name = conf.Prefix + meas_name
	}
	m := metric.New(meas_name, tags, fields, now)
	result = append(result, m)
	output.SendMetrics(result)
}

func SendDiscoveryMetrics(discovered_all int, new int, current_connected int, disconnected int, connect_error int, new_str []string, old_str []string, err_con_sids []string) {
	result := []telegraf.Metric{}

	tags := make(map[string]string)
	// and then added Extra tags from sefl-monitor config
	for k, v := range conf.ExtraLabels {
		tags[k] = v
	}

	fields := make(map[string]interface{})
	fields["all"] = discovered_all
	fields["new"] = new
	fields["current"] = current_connected
	fields["disconnected"] = disconnected
	fields["connect_errors"] = connect_error
	sort.Strings(new_str)
	sort.Strings(old_str)
	sort.Strings(err_con_sids)
	fields["disconnected_sid_names"] = strings.Join(old_str, ":")
	fields["connected_sid_names"] = strings.Join(new_str, ":")
	fields["errconnect_sid_names"] = strings.Join(new_str, ":")
	now := time.Now()
	meas_name := "discover_stats"
	if len(conf.Prefix) > 0 {
		meas_name = conf.Prefix + meas_name
	}
	m := metric.New(meas_name, tags, fields, now)
	result = append(result, m)
	output.SendMetrics(result)
}

func SendSQLDriverStat(inst string, s sql.DBStats) {
	result := []telegraf.Metric{}

	tags := make(map[string]string)

	// and then added Extra tags from sefl-monitor config
	for k, v := range conf.ExtraLabels {
		tags[k] = v
	}

	tags["instance"] = inst
	fields := make(map[string]interface{})
	fields["idle_conn"] = s.Idle
	fields["inuse_conn"] = s.InUse
	fields["max_idle_closed"] = s.MaxIdleClosed
	fields["max_idle_time_closed"] = s.MaxIdleTimeClosed
	fields["max_open_connections"] = s.MaxOpenConnections
	fields["open_connections"] = s.OpenConnections
	fields["wait_count"] = s.WaitCount
	fields["wait_duration_ms"] = s.WaitDuration.Milliseconds()

	now := time.Now()
	meas_name := "sql_driver_stats"
	if len(conf.Prefix) > 0 {
		meas_name = conf.Prefix + meas_name
	}
	m := metric.New(meas_name, tags, fields, now)
	result = append(result, m)
	output.SendMetrics(result)
}

func collectLegacyRuntimeStats() (int, error) {
	result := []telegraf.Metric{}
	nsInMs := float64(time.Millisecond)
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)

	fields := make(map[string]interface{})

	now := time.Now()
	diffTime := now.Sub(lastSampleTime).Seconds()

	fields["runtime_goroutines"] = runtime.NumGoroutine()
	fields["mem.alloc"] = memStats.Alloc
	fields["mem.mallocs"] = memStats.Mallocs
	fields["mem.frees"] = memStats.Frees
	fields["mem.sys"] = memStats.Sys

	// HEAP

	fields["mem.heap_alloc_bytes"] = memStats.HeapAlloc       // HeapAlloc is bytes of allocated heap objects.
	fields["mem.heap_sys_bytes"] = memStats.HeapSys           // HeapSys is bytes of heap memory obtained from the OS.
	fields["mem.heap_idle_bytes"] = memStats.HeapIdle         // HeapIdle is bytes in idle (unused) spans.
	fields["mem.heap_in_use_bytes"] = memStats.HeapInuse      // HeapInuse is bytes in in-use spans.
	fields["mem.heap_released_bytes"] = memStats.HeapReleased // HeapReleased is bytes of physical memory returned to the OS.
	fields["mem.heap_objects"] = memStats.HeapObjects         // HeapObjects is the number of allocated heap objects.

	// STACK/MSPAN/MCACHE

	fields["mem.stack_in_use_bytes"] = memStats.StackInuse    // StackInuse is bytes in stack spans.
	fields["mem.m_span_in_use_bytes"] = memStats.MSpanInuse   // MSpanInuse is bytes of allocated mspan structures.
	fields["mem.m_cache_in_use_bytes"] = memStats.MCacheInuse // MCacheInuse is bytes of allocated mcache structures.

	// Pause Count
	fields["gc.total_pause_ns"] = float64(memStats.PauseTotalNs) / nsInMs

	if lastPauseNs > 0 {
		pauseSinceLastSample := memStats.PauseTotalNs - lastPauseNs
		pauseInterval := float64(pauseSinceLastSample) / nsInMs
		fields["gc.pause_per_interval"] = pauseInterval
		fields["gc.pause_per_second"] = pauseInterval / diffTime
		//		log.Debugf("SELFMON:Diftime(%f) |PAUSE X INTERVAL: %f | PAUSE X SECOND %f", diffTime, pauseInterval, pauseInterval/diffTime)
	}
	lastPauseNs = memStats.PauseTotalNs

	// GC Count
	countGc := int(memStats.NumGC - lastNumGc)
	if lastNumGc > 0 {
		diff := float64(countGc)
		fields["gc.gc_per_second"] = diff / diffTime
		fields["gc.gc_per_interval"] = diff
	}
	lastNumGc = memStats.NumGC

	lastSampleTime = now

	tags := make(map[string]string)

	// and then added Extra tags from sefl-monitor config
	for k, v := range conf.ExtraLabels {
		tags[k] = v
	}
	meas_name := "runtime_gvm_stats"
	if len(conf.Prefix) > 0 {
		meas_name = conf.Prefix + meas_name
	}
	m := metric.New(meas_name, tags, fields, now)
	result = append(result, m)
	output.SendMetrics(result)
	return 0, nil
}
