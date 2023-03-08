package oracle

import (
	"time"

	"github.com/toni-moreno/oracle_collector/pkg/agent/output"
	"github.com/toni-moreno/oracle_collector/pkg/agent/selfmon"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

func ScanSystemForInstances(procPattern string, loglevel string) ([]*OracleInstance, error) {
	DetectedInstances := []*OracleInstance{}

	pf := ProcessFinder{}

	pmonfound, err := pf.FullPattern(procPattern)
	if err != nil {
		log.Errorf("[DISCOVERY]: Error on scanning system processes with pattern [%s] : Err %s", procPattern, err)
	}

	for sid, proc := range pmonfound {

		orainst := &OracleInstance{
			DiscoveredSid: sid,
			PMONpid:       proc.Pid,
		}

		DetectedInstances = append(DetectedInstances, orainst)
	}

	return DetectedInstances, nil
}

func discover(cfg *config.DiscoveryConfig) {
	oinstances, err := ScanSystemForInstances(cfg.OracleDiscoverySidRegex, cfg.OracleLogLevel)
	if err != nil {
		log.Errorf("[DISCOVERY] Error on scan instances :%s", err)
		return
	}
	log.Debugf("[DISCOVERY] System: ===========================================")
	log.Debugf("[DISCOVERY] System: Found [%d] Oracle Intances [%+v]", len(oinstances), GetSidNames(oinstances))
	existing := OraList.GetList()
	log.Debugf("[DISCOVERY] System: Existing [%d] Oracle Intances [%+v]", len(existing), GetSidNames(existing))
	new, old, same := OraList.GetNewAndOldInstances(oinstances)
	log.Debugf("[DISCOVERY] System: New (Or not previously connected) Instances Found [%d]: %+v", len(new), GetSidNames(new))
	log.Debugf("[DISCOVERY] System: Same Instances Found [%d]: %+v", len(same), GetSidNames(same))
	log.Debugf("[DISCOVERY] System: Old Registered Instances Disappeared [%d]: %+v", len(old), GetSidNames(old))

	errorConnect := 0
	errorConnectSids := []string{}
	for _, inst := range new {
		inst.cfg = cfg
		log.Infof("[DISCOVERY] New Instance found: %s", inst.DiscoveredSid)
		err := inst.Init(cfg.OracleLogLevel, cfg.OracleClusterwareEnabled, cfg.OracleStatusExtendedInfo)
		if err != nil {
			errorConnect++
			errorConnectSids = append(errorConnectSids, inst.DiscoveredSid)
			log.Errorf("[DISCOVERY] Error On Initialize Instance %s: %s", inst.DiscoveredSid, err)
			continue
		}
		OraList.Add(inst)
		output.SendMetrics(inst.GetMetrics(true))
	}
	log.Debugf("[DISCOVERY] Old Instances Found [%d]: %+v", len(old), GetSidNames(old))
	for _, inst := range old {
		log.Infof("[DISCOVERY] Instance %s is LOST", inst.DiscoveredSid)
		err := inst.End()
		if err != nil {
			log.Errorf("[DISCOVERY] Error on release Instance monitor resources for [%s]: Err: %s", inst.DiscoveredSid, err)
			break
		}
		OraList.Delete(inst)
		output.SendMetrics(inst.GetMetrics(false))
		selfmon.SendSQLDriverStat(inst.GetInstanceName(), inst.GetDriverStats())
	}
	log.Debugf("[DISCOVERY] Same Instances Found [%d]: %+v", len(same), GetSidNames(same))
	// for all other instances should update status and send metrics.

	for _, inst := range same {
		err := inst.UpdateInfo()
		if err != nil {
			log.Errorf("[DISCOVERY] Error on Update Instance Info for [%s]: Err: %s", inst.DiscoveredSid, err)
		}
		output.SendMetrics(inst.GetMetrics(true))
		selfmon.SendSQLDriverStat(inst.GetInstanceName(), inst.GetDriverStats())
	}

	selfmon.SendDiscoveryMetrics(len(oinstances), len(new), len(same), len(old), errorConnect, GetSidNames(new), GetSidNames(old), errorConnectSids)
}

func discoveryProcess(cfg *config.DiscoveryConfig, done chan bool) {
	discoveryTicker := time.NewTicker(cfg.OracleDiscoveryInterval)
	defer discoveryTicker.Stop()

	first := make(chan bool, 1)
	first <- true

	for {
		select {
		case <-first:
			log.Info("[DISCOVERY] Initializing Oracle Discovery Process....")
			discover(cfg)
		case t := <-discoveryTicker.C:
			log.Infof("[DISCOVERY] Scanning Again oracle instances at %s", t)
			discover(cfg)
		case <-done:
			return
		}
	}
}

func InitDiscovery(cfg *config.DiscoveryConfig, done chan bool) {
	go discoveryProcess(cfg, done)
}
