package oracle

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

func CreateLoggerForSid(sid string, loglevel string) *logrus.Logger {
	log := logrus.New()
	logfilename := logDir + "/collector_" + sid + ".log"
	f, _ := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	log.Out = f
	l, _ := logrus.ParseLevel(loglevel)
	log.Level = l
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.Formatter = customFormatter
	customFormatter.FullTimestamp = true
	return log
}

func ScanSystemForInstances(procPattern string, loglevel string) ([]*OracleInstance, error) {
	DetectedInstances := []*OracleInstance{}

	pf := ProcessFinder{}

	pmonfound, err := pf.FullPattern(procPattern)
	if err != nil {
		log.Error(err)
	}

	for sid, proc := range pmonfound {

		orainst := &OracleInstance{
			DiscoveredSid: sid,
			PMONpid:       proc.Pid,
			log:           CreateLoggerForSid(sid, loglevel),
		}

		DetectedInstances = append(DetectedInstances, orainst)
	}

	return DetectedInstances, nil
}

func discover(cfg *config.DiscoveryConfig) {
	oinstances, err := ScanSystemForInstances(cfg.OracleDiscoverySidRegex, cfg.OracleLogLevel)
	if err != nil {
		log.Errorf("Error on scan isntances :%s", err)
		return
	}
	for _, inst := range oinstances {
		inst.cfg = cfg
		log.Infof("Instance found: %s", inst.DiscoveredSid)
		err := inst.InitDBData()
		if err != nil {
			log.Warnf("Error On Initialize Instance %s: %s", inst.DiscoveredSid, err)
		}
	}
	OraList.SetList(oinstances)
}

func discoveryProcess(cfg *config.DiscoveryConfig, done chan bool) {
	log.Infof("Ticket %s", cfg.OracleDiscoveryInterval.String())
	discoveryTicker := time.NewTicker(cfg.OracleDiscoveryInterval)
	defer discoveryTicker.Stop()
	log.Info("Before loop")
	first := make(chan bool, 1)
	first <- true

	for {
		log.Info("Scanning oracle instances")
		select {
		case <-first:
			discover(cfg)
		case t := <-discoveryTicker.C:
			log.Infof("Scanning oracle instances at %s", t)
			discover(cfg)
		case <-done:
			return
		}
	}
}

func InitDiscovery(cfg *config.DiscoveryConfig, done chan bool) {
	go discoveryProcess(cfg, done)
}
