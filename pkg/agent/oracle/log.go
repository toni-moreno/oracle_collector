package oracle

// Debugf info
func (oi *OracleInstance) Debugf(expr string, vars ...interface{}) {
	// expr2 := "OracleInstance [" + oi.DiscoveredSid + "] " + expr
	oi.log.Debugf(expr, vars...)
}

// Infof info
func (oi *OracleInstance) Infof(expr string, vars ...interface{}) {
	// expr2 := "OracleInstance [" + oi.DiscoveredSid + "] " + expr
	oi.log.Infof(expr, vars...)
}

// Errorf info
func (oi *OracleInstance) Errorf(expr string, vars ...interface{}) {
	// expr2 := "OracleInstance [" + oi.DiscoveredSid + "] " + expr
	oi.log.Errorf(expr, vars...)
}

// Warnf log warn data
func (oi *OracleInstance) Warnf(expr string, vars ...interface{}) {
	// expr2 := "OracleInstance [" + oi.DiscoveredSid + "] " + expr
	oi.log.Warnf(expr, vars...)
}
