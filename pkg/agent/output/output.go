package output

import (
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/sirupsen/logrus"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

var (
	log *logrus.Logger
	// reloadMutex guards the reloadProcess flag
	mutex         sync.RWMutex
	bufferSize    int
	flushPeriod   time.Duration
	MetricBufChan chan *telegraf.Metric
	chExit        chan bool
)

// SetLogger sets the current log output.
func SetLogger(l *logrus.Logger) {
	log = l
}

func Init(cfg *config.OutputConfig) {
	bufferSize = cfg.BufferSize
	flushPeriod = cfg.FlushPeriod
	MetricBufChan = make(chan *telegraf.Metric, bufferSize)
	chExit = make(chan bool)
	startOutputSender()
}

// End release DB connection
func End() {
	close(MetricBufChan)
	close(chExit)
}

func SendMetrics(metrics []*telegraf.Metric) {
	for _, m := range metrics {
		MetricBufChan <- m
	}
}

func flushData() (int, error) {
	chanlen := len(MetricBufChan) // get number of entries in the batchpoint channel
	log.Infof("Flushing %d metrics of data", chanlen)
	out := []*telegraf.Metric{}
	for i := 0; i < chanlen; i++ {
		// flush them
		data := <-MetricBufChan
		out = append(out, data)
		// this process only will work if backend is  running ok elsewhere points will be lost
	}
	return outSinkArray(out)
}

func startOutputSender() {
	flushTicker := time.NewTicker(flushPeriod)
	defer flushTicker.Stop()

	log.Infof("beginning OutputSender thread")
	for {
		select {
		case <-chExit:
			// need to flush all data
			n, err := flushData()
			log.Infof("Flushed %d metrics: with error:%s", n, err)
			log.Infof("EXIT from Output sender process for device:")
			return
		case <-flushTicker.C:
			log.Debugf("Flushing data... ")
			n, err := flushData()
			log.Infof("Flushed %d metrics: with error:%s", n, err)
			// case data := <-MetricBufChan:
			// 	if data == nil {
			// 		log.Warn("null influx input")
			// 		continue
			// 	}
		}
	}
}
