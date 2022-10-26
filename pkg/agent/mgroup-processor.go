package agent

import (
	"regexp"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/toni-moreno/oracle_collector/pkg/agent/output"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

// Debugf info
func (mgp *MGroupProcessor) Debugf(expr string, vars ...interface{}) {
	expr2 := "SNMPDEVICE [" + mgp.cfg.Name + "] " + expr
	mgp.log.Debugf(expr2, vars...)
}

// Infof info
func (mgp *MGroupProcessor) Infof(expr string, vars ...interface{}) {
	expr2 := "SNMPDEVICE [" + mgp.cfg.Name + "] " + expr
	mgp.log.Infof(expr2, vars...)
}

// Errorf info
func (mgp *MGroupProcessor) Errorf(expr string, vars ...interface{}) {
	expr2 := "SNMPDEVICE [" + mgp.cfg.Name + "] " + expr
	mgp.log.Errorf(expr2, vars...)
}

// Warnf log warn data
func (mgp *MGroupProcessor) Warnf(expr string, vars ...interface{}) {
	expr2 := "SNMPDEVICE [" + mgp.cfg.Name + "] " + expr
	mgp.log.Warnf(expr2, vars...)
}

type MGroupProcessor struct {
	OracleInstances []*OracleInstance
	cfg             *config.OracleMetricGroupConfig
	log             *logrus.Logger
}

func InitGroupProcessor(cfg *config.OracleMetricGroupConfig, instances []*OracleInstance) *MGroupProcessor {
	// filter instances where apply
	var filtered []*OracleInstance

	if len(cfg.InstanceFilter) != 0 {
		for _, i := range instances {
			match, _ := regexp.MatchString(cfg.InstanceFilter, i.InstInfo.InstName)
			if match {
				filtered = append(filtered, i)
			}
		}
	} else {
		filtered = instances
	}

	ret := MGroupProcessor{
		OracleInstances: filtered,
		cfg:             cfg,
		log:             logrus.New(),
	}
	return &ret
}

func (mgp *MGroupProcessor) ProcesQuery() {
	for _, i := range mgp.OracleInstances {
		for _, q := range mgp.cfg.OracleMetrics {
			mgp.log.Infof("Query: %s in instance %s", q.Context, i.InstInfo.InstName)
			table := NewDatatableWithConfig(&q)
			n, err := i.Query(mgp.cfg.QueryTimeout, q.Request, table)
			if err != nil {
				mgp.log.Warnf("Error on query: %s", err)
				continue
			}
			mgp.log.Infof("Query: %s in instance %s: returned %d rows", q.Context, i.InstInfo.InstName, n)
			// Data transformation.
			metrics, err := table.GetMetrics()
			if err != nil {
				mgp.log.Warnf("Error on  metric transformation: %s", err)
				continue
			}
			output.SendMetrics(metrics)
		}
	}
}

func (mgp *MGroupProcessor) StartCollection(done chan bool, s *sync.WaitGroup) {
	mgp.Infof("Initializating collection process for Group (%s)", mgp.cfg.Name)
	s.Add(1)
	go func() {
		defer s.Done()

		mgp.log.Infof("Begin Query Processor for Group:  %s ( Period: %s )", mgp.cfg.Name, mgp.cfg.QueryPeriod.String())
		qTicker := time.NewTicker(mgp.cfg.QueryPeriod)
		defer qTicker.Stop()

		first := make(chan bool, 1)
		first <- true

		for {
			log.Info("Scanning oracle instances")
			select {
			case <-first:
				mgp.ProcesQuery()
			case t := <-qTicker.C:
				log.Infof("Scanning oracle instances at %s", t)
				mgp.ProcesQuery()
			case <-done:
				return
			}
		}
	}()
}
