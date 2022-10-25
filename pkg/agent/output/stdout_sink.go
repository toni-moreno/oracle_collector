package output

import "github.com/influxdata/telegraf"

func outSink(m *telegraf.Metric) {
	log.Infof("SINK METRIC: %+v", m)
}

func outSinkArray(ma []*telegraf.Metric) (int, error) {
	for _, m := range ma {
		log.Infof("SINK METRIC: %+v", m)
	}
	return len(ma), nil
}
