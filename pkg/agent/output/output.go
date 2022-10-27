package output

import (
	"bufio"
	"os"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/models"
	"github.com/influxdata/telegraf/plugins/serializers"
	"github.com/sirupsen/logrus"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

var (
	log         *logrus.Logger
	bufferSize  int
	bachSize    int
	flushPeriod time.Duration
	// MetricBufChan chan *telegraf.Metric
	chExit chan bool
	buffer *models.Buffer
	ser    serializers.Serializer
)

// SetLogger sets the current log output.
func SetLogger(l *logrus.Logger) {
	log = l
}

func Init(cfg *config.OutputConfig) {
	var err error
	sc := &serializers.Config{TimestampUnits: 1 * time.Second}
	sc.DataFormat = "influx"
	ser, err = serializers.NewSerializer(sc)
	if err != nil {
		log.Error("Error in init serializer")
	}

	buffer = models.NewBuffer("oracle_collector", "buff", cfg.BufferSize)
	bufferSize = cfg.BufferSize
	bachSize = cfg.BatchSize
	flushPeriod = cfg.FlushPeriod
	// MetricBufChan = make(chan *telegraf.Metric, bufferSize)
	chExit = make(chan bool)
	go startOutputSender()
}

// End release DB connection
func End() {
	// close(MetricBufChan)
	close(chExit)
}

func SendMetrics(metrics []telegraf.Metric) {
	dropped := buffer.Add(metrics...)
	if dropped > 0 {
		log.Warnf("Dropped metrics %d", dropped)
	}
	// for _, m := range metrics {
	// 	MetricBufChan <- m
	// }
}

func flushData() (int, error) {
	// chanlen := len(MetricBufChan) // get number of entries in the batchpoint channel
	// log.Infof("Flushing %d metrics of data", chanlen)
	// out := []*telegraf.Metric{}
	// for i := 0; i < chanlen; i++ {
	// 	// flush them
	// 	data := <-MetricBufChan
	// 	out = append(out, data)
	// 	// this process only will work if backend is  running ok elsewhere points will be lost
	// }
	out := buffer.Batch(buffer.Len())
	outbytes, _ := ser.SerializeBatch(out)
	f := bufio.NewWriter(os.Stdout)
	defer f.Flush()
	f.Write(outbytes)
	// return outSinkArray(out)
	return len(out), nil
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
		}
	}
}
