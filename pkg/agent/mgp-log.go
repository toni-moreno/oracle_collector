package agent

import "github.com/toni-moreno/oracle_collector/pkg/agent/oracle"

// Debugf info
func (mgp *MGroupProcessor) Debugf(i *oracle.OracleInstance, expr string, vars ...interface{}) {
	expr2 := "GroupProc [" + mgp.cfg.Name + "] " + expr
	i.Debugf(expr2, vars...)
}

// Infof info
func (mgp *MGroupProcessor) Infof(i *oracle.OracleInstance, expr string, vars ...interface{}) {
	expr2 := "GroupProc[" + mgp.cfg.Name + "] " + expr
	i.Infof(expr2, vars...)
}

// Errorf info
func (mgp *MGroupProcessor) Errorf(i *oracle.OracleInstance, expr string, vars ...interface{}) {
	expr2 := "GroupProc [" + mgp.cfg.Name + "] " + expr
	i.Errorf(expr2, vars...)
}

// Warnf log warn data
func (mgp *MGroupProcessor) Warnf(i *oracle.OracleInstance, expr string, vars ...interface{}) {
	expr2 := "GroupProc [" + mgp.cfg.Name + "] " + expr
	i.Warnf(expr2, vars...)
}

func (mgp *MGroupProcessor) BroadCastInfof(expr string, vars ...interface{}) {
	for _, i := range mgp.OracleInstances {
		expr2 := "GroupProc[" + mgp.cfg.Name + "] " + expr
		i.Infof(expr2, vars...)
	}
}
