package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"

	_ "github.com/godror/godror"
	"github.com/sirupsen/logrus"

	"github.com/hashicorp/go-version"
	"github.com/toni-moreno/oracle_collector/pkg/agent/data"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

// https://docs.oracle.com/database/121/REFRN/GUID-A399F608-36C8-4DF0-9A13-CEE25637653E.htm#REFRN30652
type PdbInfo struct {
	ConID          int
	Name           string
	OpenMode       string
	Restricted     string
	RecoveryStatus string
	TotalSize      int
	BlockSize      int
}

func (pi *PdbInfo) GetMetric(extralabels map[string]string) telegraf.Metric {
	tags := make(map[string]string)
	// first added extra tags
	for k, v := range extralabels {
		tags[k] = v
	}
	tags["pdb_name"] = pi.Name
	fields := make(map[string]interface{})

	fields["open_mode"] = pi.OpenMode
	fields["restricted"] = pi.Restricted
	fields["recovery_status"] = pi.RecoveryStatus
	fields["total_size"] = pi.TotalSize
	fields["block_size"] = pi.BlockSize
	return metric.New("oracle_pdb_status", tags, fields, time.Now())
}

type InstanceInfo struct {
	InstNumber      int
	InstName        string
	HostName        string
	Version         string
	StartupTime     string
	Uptime          string
	Status          string
	DBStatus        string
	InstanceRole    string
	ActiveState     string
	Bloqued         string
	ShutdownPending string
	Archiver        string
}

type DatabaseInfo struct {
	DBID         string
	DbName       string
	Created      string
	DBUniqName   string
	CDB          string
	OpenMode     string
	DatabaseRole string
	ForceLogging string
	LogMode      string
	PDBs         []PdbInfo
	PDBTotal     int
	PDBActive    int
}

type OracleInstance struct {
	sync.Mutex
	DiscoveredSid string
	// Instance Info
	InitVersion *version.Version // First version check

	InstInfo InstanceInfo
	DBInfo   DatabaseInfo

	ClusteWareEnabled bool
	IsValidForDBQuery bool

	AlertLogFile string
	ListenerIP   string
	ListenerPort int
	PMONpid      int32
	cfg          *config.DiscoveryConfig
	conn         *sql.DB
	log          *logrus.Logger
	labels       map[string]string
}

func (oi *OracleInstance) String() string {
	return fmt.Sprintf("[ Discovered %s,  INSTANCE INFO: [%+v], DBInfo [%+v]", oi.DiscoveredSid, oi.InstInfo, oi.DBInfo)
}

func (oi *OracleInstance) GetExtraLabels() map[string]string {
	oi.Lock()
	defer oi.Unlock()
	return oi.labels
}

func (oi *OracleInstance) GetIsValidForDBQuery() bool {
	oi.Lock()
	defer oi.Unlock()
	return oi.IsValidForDBQuery
}

func (oi *OracleInstance) GetInstanceName() string {
	oi.Lock()
	defer oi.Unlock()
	return oi.InstInfo.InstName
}

func (oi *OracleInstance) GetDriverStats() sql.DBStats {
	oi.Lock()
	defer oi.Unlock()
	return oi.conn.Stats()
}

func (oi *OracleInstance) initExtraLabels() map[string]string {
	oi.labels = make(map[string]string)
	// First fixed labels
	for k, v := range oi.cfg.ExtraLabels {
		oi.labels[k] = v
	}
	// Dinamic labels.
	SID := oi.InstInfo.InstName // oi.Discovered SID

	for n, rule := range oi.cfg.DynamicParamsBySID {
		oi.log.Debugf("EXTRA LABELS: Applying rule [%d] info with sid_regex = %s", n, rule.SidRegex)
		match := rule.R.MatchString(SID)
		if match {
			for k, v := range rule.ExtraLabels {
				oi.labels[k] = v
			}
		}
	}
	// Oracle Mandatory Labels.

	oi.labels["instance"] = oi.InstInfo.InstName
	// labels["instance_num"] = strconv.Itoa(oi.InstInfo.InstNumber)
	oi.labels["instance_role"] = oi.InstInfo.InstanceRole
	oi.labels["db"] = oi.DBInfo.DbName
	// labels["db_unique_name"] = oi.DBInfo.DBUniqName
	return oi.labels
}

func (oi *OracleInstance) Query(timeout time.Duration, query string, t *data.DataTable) (int, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	rows, err := oi.conn.QueryContext(ctx, query) // DATA RACE FOUND
	if ctx.Err() == context.DeadlineExceeded {
		return 0, 0, errors.New("Oracle query timed out")
	}
	if err != nil {
		elapsed := time.Since(start)
		return 0, elapsed, fmt.Errorf("Error in instance Query:%s", err)
	}
	defer rows.Close()
	c, err := rows.Columns()
	if err != nil {
		elapsed := time.Since(start)
		return 0, elapsed, fmt.Errorf("Error on Query Columns:%s", err)
	}
	t.SetHeader(c)

	for rows.Next() {
		rowpointers := t.AppendEmptyRow()
		if err := rows.Scan(rowpointers...); err != nil {
			elapsed := time.Since(start)
			return 0, elapsed, err
		}
	}
	elapsed := time.Since(start)
	return t.Length(), elapsed, nil
}

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

func (oi *OracleInstance) GetVersion() error {
	var err error
	oi.Lock()
	defer oi.Unlock()
	log.Infof("[DISCOVERY] Get Version Instance Info...")
	query := "select VERSION from V$INSTANCE"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	row := oi.conn.QueryRowContext(ctx, query)

	var ver string
	err = row.Scan(&ver)
	if err != nil {
		return err
	}
	oi.InitVersion, err = version.NewVersion(ver)
	if err != nil {
		return fmt.Errorf("Error on format version %s", err)
	}

	return nil
}

func (oi *OracleInstance) CheckVersionGreaterThanOrEqual(ver string) (string, bool) {
	oi.Lock()
	defer oi.Unlock()
	v, _ := version.NewVersion(ver)
	return oi.InitVersion.String(), oi.InitVersion.GreaterThanOrEqual(v)
}

func (oi *OracleInstance) CheckVersionLessThan(ver string) (string, bool) {
	oi.Lock()
	defer oi.Unlock()
	v, _ := version.NewVersion(ver)
	return oi.InitVersion.String(), oi.InitVersion.LessThan(v)
}

func (oi *OracleInstance) CheckVersionBetween(lower_ver, upper_ver string) (string, bool) {
	oi.Lock()
	defer oi.Unlock()
	l, _ := version.NewVersion(lower_ver)
	u, _ := version.NewVersion(upper_ver)
	return oi.InitVersion.String(), (oi.InitVersion.GreaterThanOrEqual(l) && oi.InitVersion.LessThan(u))
}

func (oi *OracleInstance) UpdateInfo() error {
	oi.Lock()
	defer oi.Unlock()
	// Initialize instance Data.
	// tested on 11.2.0.4.0/12.1.0.2.0/19.7.0.0.0
	log.Infof("[DISCOVERY] Initialize/Update Instance Info...")
	// VERSION_FULL as VERSION not working on 11.2.0.4
	var query string
	v18c, _ := version.NewVersion("12.2.0.2")
	switch {
	// https://oracle-base.com/articles/18c/articles-18c
	// https://docs.oracle.com/en/database/oracle/oracle-database/19/refrn/V-INSTANCE.html
	case oi.InitVersion.GreaterThan(v18c): // 18c
		query = `
				select 
					INSTANCE_NUMBER,
					INSTANCE_NAME,
					HOST_NAME,
					VERSION_FULL AS VERSION,
					STARTUP_TIME,
					FLOOR((SYSDATE - STARTUP_TIME) * 60 * 60 * 24) as UPTIME,
					STATUS,
					DATABASE_STATUS,
					INSTANCE_ROLE,
					ACTIVE_STATE,
					BLOCKED,
					SHUTDOWN_PENDING,
					ARCHIVER
				from V$INSTANCE
				`
	default:
		query = `
				select 
					INSTANCE_NUMBER,
					INSTANCE_NAME,
					HOST_NAME,
					VERSION,
					STARTUP_TIME,
					FLOOR((SYSDATE - STARTUP_TIME) * 60 * 60 * 24) as UPTIME,
					STATUS,
					DATABASE_STATUS,
					INSTANCE_ROLE,
					ACTIVE_STATE,
					BLOCKED,
					SHUTDOWN_PENDING,
					ARCHIVER
				from V$INSTANCE
		`
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows_i, err := oi.conn.QueryContext(ctx, query)
	if ctx.Err() == context.DeadlineExceeded {
		return errors.New("Oracle Info query timed out")
	}
	if err != nil {
		log.Warnf("[DISCOVERY] Error in instance Query:%s", err)
		return err
	}
	defer rows_i.Close()
	rowsCount := 0
	for rows_i.Next() {
		err = rows_i.Scan(
			&oi.InstInfo.InstNumber,
			&oi.InstInfo.InstName,
			&oi.InstInfo.HostName,
			&oi.InstInfo.Version,
			&oi.InstInfo.StartupTime,
			&oi.InstInfo.Uptime,
			&oi.InstInfo.Status,
			&oi.InstInfo.DBStatus,
			&oi.InstInfo.InstanceRole,
			&oi.InstInfo.ActiveState,
			&oi.InstInfo.Bloqued,
			&oi.InstInfo.ShutdownPending,
			&oi.InstInfo.Archiver,
		)
		if err != nil {
			return err
		}
		rowsCount += 1
	}
	log.Debugf("[DISCOVERY] Instance Rows:%d", rowsCount)
	oi.InitVersion, _ = version.NewVersion(oi.InstInfo.Version)

	// https://www.oracletutorial.com/oracle-administration/oracle-startup/
	// ------------------------------------------------
	// NOMOUNT => INST ( STARTED ) => DB  (N.A)
	// MOUNT => INST(MOUNTED)=> DB MOUNTED
	// OPEN => INTT(OPEN) => DB READ WRITE
	// Initialize DB Data Only if instance not in STARTED mode.
	if oi.InstInfo.Status == "STARTED" {
		return nil
	}
	log.Infof("[DISCOVERY] Initialize/Update Database Info...")
	query = `
		select 
			DBID,
			NAME,
			CREATED,
			DB_UNIQUE_NAME,
			OPEN_MODE,
			DATABASE_ROLE,
			FORCE_LOGGING,
			LOG_MODE 
		from v$database`
	rows_db, err := oi.conn.QueryContext(ctx, query)
	if err != nil {
		log.Warnf("[DISCOVERY] Error in database Query:%s", err)
		return err
	}
	defer rows_db.Close()
	rowsCount = 0
	for rows_db.Next() {
		err = rows_db.Scan(
			&oi.DBInfo.DBID,
			&oi.DBInfo.DbName,
			&oi.DBInfo.Created,
			&oi.DBInfo.DBUniqName,
			&oi.DBInfo.OpenMode,
			&oi.DBInfo.DatabaseRole,
			&oi.DBInfo.ForceLogging,
			&oi.DBInfo.LogMode,
		)
		if err != nil {
			return err
		}
		rowsCount += 1
	}
	log.Debugf("[DISCOVERY] DB Rows:%d", rowsCount)
	// Check if this instance will be useful for DB level queries
	// if ClusterWare Mode Enabled
	//  by default disabled
	oi.IsValidForDBQuery = false
	if oi.ClusteWareEnabled {
		query = `SELECT MIN(INSTANCE_NUMBER) AS MIN FROM GV$INSTANCE`
		row := oi.conn.QueryRowContext(ctx, query)
		var min int
		rowsCount = 0

		err = row.Scan(&min)
		if err != nil {
			return err
		}

		log.Debugf("[DISCOVERY] Cluster (MIN: %d)", min)
		if oi.InstInfo.InstNumber == min {
			log.Debugf("This instance %s[%d] is the lowest instance => Is Valid for DB Queries ", oi.InstInfo.InstName, oi.InstInfo.InstNumber)
			oi.IsValidForDBQuery = true
		}
	} else {
		// all db queries shoud be done also if clusterware not enabled
		oi.IsValidForDBQuery = true
	}

	// Initialice PDB's info.
	log.Infof("[DISCOVERY] Initialize/Update PDB Info...")
	oi.DBInfo.PDBs = nil
	// https://docs.oracle.com/database/121/REFRN/GUID-A399F608-36C8-4DF0-9A13-CEE25637653E.htm#REFRN30652

	v12c, _ := version.NewVersion("12.1")
	v12_1_0_2, _ := version.NewVersion("12.1.0.2")
	rowsCount = 0
	activeCount := 0
	switch {
	case oi.InitVersion.LessThan(v12c):
		// nothing to do: not supported PDB's
	case oi.InitVersion.LessThan(v12_1_0_2):
		query = `
					select 
						CON_ID,
						NAME,
						OPEN_MODE,
						RESTRICTED,
						TOTAL_SIZE,
					from v$pdbs
					`
		rows_pdb, err := oi.conn.QueryContext(ctx, query)
		if err != nil {
			log.Warnf("[DISCOVERY] Error in PDB Query:%s", err)
			return err
		}
		defer rows_pdb.Close()

		for rows_pdb.Next() {
			pdb := PdbInfo{}
			err = rows_pdb.Scan(
				&pdb.ConID,
				&pdb.Name,
				&pdb.OpenMode,
				&pdb.Restricted,
				&pdb.TotalSize,
			)

			if err != nil {
				return err
			}
			rowsCount += 1
			oi.DBInfo.PDBs = append(oi.DBInfo.PDBs, pdb)
			if pdb.OpenMode == "READ WRITE" {
				activeCount++
			}
		}
	default:
		query = `
					select 
						CON_ID,
						NAME,
						OPEN_MODE,
						RESTRICTED,
						RECOVERY_STATUS,
						TOTAL_SIZE,
						BLOCK_SIZE
					from v$pdbs
					`
		rows_pdb, err := oi.conn.QueryContext(ctx, query)
		if err != nil {
			log.Warnf("[DISCOVERY] Error in PDB Query:%s", err)
			return err
		}
		defer rows_pdb.Close()

		for rows_pdb.Next() {
			pdb := PdbInfo{}
			err = rows_pdb.Scan(
				&pdb.ConID,
				&pdb.Name,
				&pdb.OpenMode,
				&pdb.Restricted,
				&pdb.RecoveryStatus,
				&pdb.TotalSize,
				&pdb.BlockSize,
			)

			if err != nil {
				return err
			}
			rowsCount += 1
			oi.DBInfo.PDBs = append(oi.DBInfo.PDBs, pdb)
			if pdb.OpenMode == "READ WRITE" {
				activeCount++
			}
		}
	}

	oi.DBInfo.PDBTotal = rowsCount
	oi.DBInfo.PDBActive = activeCount
	log.Debugf("[DISCOVERY] PDB's Rows:%d, Active %d", rowsCount, activeCount)
	log.Debugf("[DISCOVERY] Found %s", oi) // DATA RACE DETECTED
	oi.initExtraLabels()
	return nil
}

func (oi *OracleInstance) Init(loglevel string, ClusterwareEnabled bool) error {
	var err error

	oi.ClusteWareEnabled = ClusterwareEnabled

	oi.log = CreateLoggerForSid(oi.DiscoveredSid, loglevel)

	// Get Initializacion parametres

	ConnectDSN := oi.cfg.OracleConnectDSN
	ConnectUser := oi.cfg.OracleConnectUser
	ConnectPass := oi.cfg.OracleConnectPass

	for n, rule := range oi.cfg.DynamicParamsBySID {
		log.Debugf("ORACLE INIT: Applying rule [%d] info with sid_regex = %s", n, rule.SidRegex)
		match := rule.R.MatchString(oi.DiscoveredSid)
		if match {
			log.Infof("ORACLE INIT: Dinamic params match at rule %d:[%s] ", n, rule.SidRegex)
			if len(rule.OracleConnectDSN) > 0 {
				ConnectDSN = rule.OracleConnectDSN
			}
			if len(rule.OracleConnectUser) > 0 {
				ConnectUser = rule.OracleConnectUser
			}
			if len(rule.OracleConnectPass) > 0 {
				ConnectPass = rule.OracleConnectPass
			}
		}
	}

	dsn := strings.ReplaceAll(ConnectDSN, "SID", oi.DiscoveredSid)
	connStr := "oracle://" + url.QueryEscape(ConnectUser) + ":" + url.QueryEscape(ConnectPass) + "@" + dsn
	oi.Lock()
	oi.conn, err = sql.Open("godror", connStr)
	if err != nil {
		log.Warnf("[DISCOVERY] Can't create connection: %s ", err)
		oi.Unlock()
		return fmt.Errorf("ConnectDNS: %s: ERR: %s", dsn, err)
	}
	log.Tracef("[DISCOVERY] Connection String: %s", connStr)
	oi.conn.SetConnMaxLifetime(0)
	oi.conn.SetMaxIdleConns(10)
	oi.conn.SetMaxOpenConns(10)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Connection Ping
	err = oi.conn.PingContext(ctx)
	if ctx.Err() == context.DeadlineExceeded {
		oi.Unlock()
		return fmt.Errorf("ConnectDNS: %s: Oracle Ping timed out")
	}
	if err != nil {
		log.Warnf("[DISCOVERY] Can't ping connection: %s ", err)
		oi.Unlock()
		return fmt.Errorf("ConnectDNS: %s: ERR: %s", dsn, err)
	}
	oi.Unlock()
	oi.GetVersion()
	return oi.UpdateInfo()
}

func (oi *OracleInstance) End() error {
	oi.Lock()
	defer oi.Unlock()
	err := oi.conn.Close()
	if err != nil {
		log.Errorf("[DISCOVERY] Error while closing oracle connection: %s:", err)
	}
	return nil
}

func (oi *OracleInstance) StatusMetrics(process_ok bool) []telegraf.Metric {
	tags := make(map[string]string)
	// first added extra tags
	for k, v := range oi.labels {
		tags[k] = v
	}
	fields := make(map[string]interface{})

	fields["proc_ok"] = process_ok
	fields["proc_pid"] = oi.PMONpid
	fields["inst_active_state"] = oi.InstInfo.ActiveState
	fields["inst_bloqued"] = oi.InstInfo.Bloqued
	fields["inst_db_status"] = oi.InstInfo.DBStatus
	fields["inst_number"] = oi.InstInfo.InstNumber
	fields["inst_status"] = oi.InstInfo.Status
	fields["inst_startup_time"] = oi.InstInfo.StartupTime
	fields["inst_uptime"] = oi.InstInfo.Uptime
	fields["inst_version"] = oi.InstInfo.Version
	fields["inst_role"] = oi.InstInfo.InstanceRole
	fields["inst_shutdown_pending"] = oi.InstInfo.ShutdownPending
	fields["inst_archiver"] = oi.InstInfo.Archiver
	fields["db_open_mode"] = oi.DBInfo.OpenMode
	fields["db_created"] = oi.DBInfo.Created
	fields["db_role"] = oi.DBInfo.DatabaseRole
	fields["db_log_mode"] = oi.DBInfo.LogMode
	fields["db_force_logging"] = oi.DBInfo.ForceLogging
	fields["pdbs_total"] = oi.DBInfo.PDBTotal
	fields["pdbs_active"] = oi.DBInfo.PDBActive

	instance_ok := 0

	// 0 = not process
	// 1 = instance up not mounted
	// 2 = instance up mounted
	// 3 = instance up and open db readpnñy
	// 4 = instance up adn open in rw with performance errors
	// 5 = instance up adn open in rw without performance errors

	// https://docs.oracle.com/cd/B19306_01/server.102/b14237/dynviews_1131.htm#REFRN30105
	switch oi.InstInfo.Status {
	case "STARTED": // STARTED = 1
		instance_ok = 1
	case "MOUNTED": // MOUNTED = 2
		instance_ok = 2
	case "OPEN": // OPEN = 3
		switch oi.DBInfo.OpenMode {
		case "MOUNTED":
			instance_ok = 3
		case "READ WRITE":
			instance_ok = 5
		case "READ ONLY":
			instance_ok = 3
		default:
			log.Errorf("Error on DB OpenMode %s", oi.DBInfo.OpenMode)
		}
	case "OPEN MIGRATE": // OPEN MIGRATE = 2
		instance_ok = 2 // 3?
	default:
		log.Errorf("Error on Instance Active State %s", oi.InstInfo.ActiveState)
	}

	fields["instance_ok"] = instance_ok
	status := metric.New("oracle_status", tags, fields, time.Now())

	return []telegraf.Metric{status}
}

func (oi *OracleInstance) GetMetrics(process_ok bool) []telegraf.Metric {
	var ret []telegraf.Metric

	im := oi.StatusMetrics(process_ok)
	ret = append(ret, im...)
	// Sending pdb metrics only if valid for db queries
	// ( single instances or clusterware enabled and vaildforDBQuery's check is true)

	if oi.IsValidForDBQuery {
		for _, v := range oi.DBInfo.PDBs {
			pm := v.GetMetric(oi.labels)
			ret = append(ret, pm)
		}
	}

	return ret
}
