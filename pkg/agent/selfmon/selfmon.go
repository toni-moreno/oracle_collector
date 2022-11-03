package selfmon

import (
	"path/filepath"
	"runtime/metrics"
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
	// mutex  sync.Mutex
)

// SetLogger set log SELF_MON
func SetLogger(l *logrus.Logger) {
	log = l
}

func Init(cfg *config.SelfMonConfig) {
	conf = cfg
	chExit = make(chan bool)
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

func collectRuntimeData() (int, error) {
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

		meas_name := strings.ReplaceAll(utils.TrimLeftChar(filepath.Dir(sample.Name)), "/", "_")
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
			n, err := collectRuntimeData()
			log.Infof("[SELF_MON] Flushed %d metrics: with error:%s", n, err)
			log.Infof("[SELF_MON] EXIT from SELF_MON sender process for device:")
			return
		case <-flushTicker.C:
			n, err := collectRuntimeData()
			if err != nil {
				log.Infof("[SELF_MON] Flushed %d metrics: with error:%s", n, err)
			} else {
				log.Infof("[SELF_MON] Flushed %d metrics: without error", n)
			}

		}
	}
}
