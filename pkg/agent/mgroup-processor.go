package agent

import (
	"sync"
	"time"

	"github.com/toni-moreno/oracle_collector/pkg/agent/data"
	"github.com/toni-moreno/oracle_collector/pkg/agent/oracle"
	"github.com/toni-moreno/oracle_collector/pkg/agent/output"
	"github.com/toni-moreno/oracle_collector/pkg/agent/selfmon"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

type MGroupProcessor struct {
	InstanceList    *oracle.InstanceList
	OracleInstances []*oracle.OracleInstance
	cfg             *config.OracleMetricGroupConfig
	InstNames       []string
}

func InitGroupProcessor(cfg *config.OracleMetricGroupConfig, oralist *oracle.InstanceList) *MGroupProcessor {
	ret := MGroupProcessor{
		InstanceList: oralist,
		cfg:          cfg,
	}
	return &ret
}

func (mgp *MGroupProcessor) UpdateInstances() int {
	mgp.InstNames = nil
	var filtered []*oracle.OracleInstance
	// TODO: review if needed filter Instances by instance STATUS (OPEN)??
	// may some configurable queries could be donde with instance status (MOUNT)??
	instances := mgp.InstanceList.GetList()
	if len(mgp.cfg.InstanceFilter) != 0 {
		filtered = mgp.InstanceList.GetFilteredListBySid(mgp.cfg.InstanceFilter)
	} else {
		filtered = instances
		for _, i := range instances {
			mgp.InstNames = append(mgp.InstNames, i.InstInfo.InstName)
		}
	}
	mgp.OracleInstances = filtered
	mgp.InstNames = oracle.GetSidNames(mgp.OracleInstances)
	ntotal := len(instances)
	nfilter := len(filtered)
	log.Debugf("[COLLECTOR] On update Number instances total [%d] Filtered [%d]", ntotal, nfilter)
	return nfilter
}

func (mgp *MGroupProcessor) ProcesQuery() {
	n := mgp.UpdateInstances()
	mgp.BroadCastInfof("Init Query Process on [%d] Instances [%+v] ", n, mgp.InstNames)

	log.Infof("[COLLECTOR] Processor [%s] new Iteration on [%d] Instances [%+v]", mgp.cfg.Name, n, mgp.InstNames)
	for _, i := range mgp.OracleInstances {
		// check if this instance should be queried
		if mgp.cfg.GetQueryLevel() == "db" && !i.GetIsValidForDBQuery() {
			mgp.Infof(i, "QUERY IN DB MODE: SKIP querying instance %s : not smalest Instance in DB (Current %d)", i.InstInfo.InstName, i.InstInfo.InstNumber)
			continue
		}
		extraLabels := i.GetExtraLabels()
		for _, q := range mgp.cfg.OracleMetrics {
			mgp.Debugf(i, "Begin Metric Query: [%s]", q.Context)
			table := data.NewDatatableWithConfig(&q)
			n, d, err := i.Query(mgp.cfg.QueryTimeout, q.Request, table)
			if err != nil {
				mgp.Errorf(i, "Error on query: %s (Duration: %s)", err, d)
				continue
			}
			mgp.Infof(i, "Oracle Metric Query: [%s] returned [%d] rows (Transposed by: %s)(Duration: %s)", q.Context, n, q.FieldToAppend, d)
			// Data transformation.
			metrics, err := table.GetMetrics(extraLabels)
			if err != nil {
				mgp.Warnf(i, "Oracle Metric Query: [%s] Error on  metric transformation: %s", q.Context, err)
				continue
			}
			output.SendMetrics(metrics)
			selfmon.SendQueryStat(extraLabels, mgp.cfg, &q, n, d)
		}
	}
}

func (mgp *MGroupProcessor) StartCollection(done chan bool, s *sync.WaitGroup) {
	s.Add(1)
	go func() {
		defer s.Done()

		qTicker := time.NewTicker(mgp.cfg.QueryPeriod)
		defer qTicker.Stop()

		first := make(chan bool, 1)
		first <- true

		for {
			select {
			case <-first:
				log.Infof("[COLLECTOR] Start Query Processor for Group:  %s ( Period: %s )", mgp.cfg.Name, mgp.cfg.QueryPeriod.String())
				mgp.ProcesQuery()
			case <-qTicker.C:
				mgp.ProcesQuery()
			case <-done:
				return
			}
		}
	}()
}
