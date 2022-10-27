package agent

import (
	"regexp"
	"sync"
	"time"

	"github.com/toni-moreno/oracle_collector/pkg/agent/data"
	"github.com/toni-moreno/oracle_collector/pkg/agent/oracle"
	"github.com/toni-moreno/oracle_collector/pkg/agent/output"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

type MGroupProcessor struct {
	InstanceList    *oracle.InstanceList
	OracleInstances []*oracle.OracleInstance
	cfg             *config.OracleMetricGroupConfig
}

func InitGroupProcessor(cfg *config.OracleMetricGroupConfig, oralist *oracle.InstanceList) *MGroupProcessor {
	ret := MGroupProcessor{
		InstanceList: oralist,
		cfg:          cfg,
	}
	return &ret
}

func (mgp *MGroupProcessor) UpdateInstances() int {
	var filtered []*oracle.OracleInstance
	instances := mgp.InstanceList.GetList()
	if len(mgp.cfg.InstanceFilter) != 0 {
		for _, i := range instances {
			match, _ := regexp.MatchString(mgp.cfg.InstanceFilter, i.InstInfo.InstName)
			if match {
				filtered = append(filtered, i)
			}
		}
	} else {
		filtered = instances
	}
	mgp.OracleInstances = filtered
	return len(mgp.OracleInstances)
}

func (mgp *MGroupProcessor) ProcesQuery() {
	n := mgp.UpdateInstances()
	log.Infof("Number of instances found %d", n)
	for _, i := range mgp.OracleInstances {
		for _, q := range mgp.cfg.OracleMetrics {
			mgp.Infof(i, "Query: %s in instance %s", q.Context, i.InstInfo.InstName)
			table := data.NewDatatableWithConfig(&q)
			n, err := i.Query(mgp.cfg.QueryTimeout, q.Request, table)
			if err != nil {
				i.Warnf("Error on query: %s", err)
				continue
			}
			mgp.Infof(i, "Oracle Metric Query: [%s] returned [%d] rows", q.Context, n)
			// Data transformation.
			metrics, err := table.GetMetrics()
			if err != nil {
				mgp.Warnf(i, "Error on  metric transformation: %s", err)
				continue
			}
			output.SendMetrics(metrics)
		}
	}
}

func (mgp *MGroupProcessor) StartCollection(done chan bool, s *sync.WaitGroup) {
	log.Infof("Initializating collection process for Group (%s)", mgp.cfg.Name)
	s.Add(1)
	go func() {
		defer s.Done()

		log.Infof("Begin Query Processor for Group:  %s ( Period: %s )", mgp.cfg.Name, mgp.cfg.QueryPeriod.String())
		qTicker := time.NewTicker(mgp.cfg.QueryPeriod)
		defer qTicker.Stop()

		first := make(chan bool, 1)
		first <- true

		for {
			log.Info("Begin Query Processor")
			select {
			case <-first:
				mgp.ProcesQuery()
			case t := <-qTicker.C:
				log.Infof("Begin Query Processor %s", t)
				mgp.ProcesQuery()
			case <-done:
				return
			}
		}
	}()
}
