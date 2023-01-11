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
	chExit      chan bool
	buffer      *models.Buffer
	ser         serializers.Serializer
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
		log.Error("[OUTPUT] Error in init serializer")
	}
	bufferSize = cfg.BufferSize
	bachSize = cfg.BatchSize
	flushPeriod = cfg.FlushPeriod
	buffer = models.NewBuffer("oracle_collector", "buff", cfg.BufferSize)
	chExit = make(chan bool)
	go startOutputSender()
}

// End release Output process
func End() {
	close(chExit)
}

func SendMetrics(metrics []telegraf.Metric) {
	dropped := buffer.Add(metrics...)
	if dropped > 0 {
		log.Warnf("[OUTPUT]  Dropped metrics %d", dropped)
	}
}

func flushData() (int, error) {
	out := buffer.Batch(buffer.Len())
	outbytes, _ := ser.SerializeBatch(out)
	f := bufio.NewWriter(os.Stdout)
	defer f.Flush()
	f.Write(outbytes)
	return len(out), nil
}

func startOutputSender() {
	flushTicker := time.NewTicker(flushPeriod)
	defer flushTicker.Stop()

	log.Infof("[OUTPUT] beginning OutputSender thread")
	for {
		select {
		case <-chExit:
			// need to flush all data
			n, err := flushData()
			log.Infof("[OUTPUT] Flushed %d metrics: with error:%s", n, err)
			log.Infof("[OUTPUT] EXIT from Output sender process for device:")
			return
		case <-flushTicker.C:
			n, err := flushData()
			if err != nil {
				log.Infof("[OUTPUT] Flushed %d metrics: with error:%s", n, err)
			} else {
				log.Infof("[OUTPUT] Flushed %d metrics: without error", n)
			}

		}
	}
}
