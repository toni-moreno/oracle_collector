package selfmon

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

var (
	log    *logrus.Logger
	conf   *config.SelfMonConfig
	TagMap map[string]string
	chExit chan bool
	mutex  sync.Mutex
)

// SetLogger set log SELF_MON
func SetLogger(l *logrus.Logger) {
	log = l
}

func Init(cfg *config.SelfMonConfig) {
	conf = cfg
	chExit = make(chan bool)
	go startSelfmonCollector()
}

// End release SELF_MON process
func End() {
	close(chExit)
}

func collectRuntimeData() (int, error) {
	return 0, nil
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
