package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"

	_ "github.com/godror/godror"
	"github.com/sirupsen/logrus"

	"github.com/toni-moreno/oracle_collector/pkg/agent/data"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

// https://docs.oracle.com/database/121/REFRN/GUID-A399F608-36C8-4DF0-9A13-CEE25637653E.htm#REFRN30652
type PdbInfo struct {
	ConID     int
	Name      string
	OpenMode  string
	Resticted string
}

type InstanceInfo struct {
	InstNumber      int    `db:"name:INSTANCE_NUMBER"`
	InstName        string `db:"name:INSTANCE_NAME"`
	HostName        string `db:"name:HOST_NAME"`
	Version         string `db:"name:VERSION"`
	StartupTime     string `db:"name:STARTUP_TIME"`
	Uptime          string `db:"name:UPTIME"`
	Status          string `db:"name:STATUS"`
	DBStatus        string `db:"name:DATABASE_STATUS"`
	InstanceRole    string `db:"name:INSTANCE_ROLE"`
	ActiveState     string `db:"name:ACTIVE_STATE"`
	Bloqued         string `db:"name:BLOCKED"`
	ShutdownPending string `db:"name:SHUTDOWN_PENDING"`
	Archiver        string `db:"name:ARCHIVER"`
}

type DatabaseInfo struct {
	DBID         string    `db:"name:DBID"`
	DbName       string    `db:"name:NAME"`
	Created      string    `db:"name:CREATED"`
	DBUniqName   string    `db:"name:DB_UNIQUE_NAME"`
	CDB          string    `db:"name:CDB"`
	OpenMode     string    `db:"name:OPEN_MODE"`
	DatabaseRole string    `db:"name:DATABASE_ROLE"`
	ForceLogging string    `db:"name:FORCE_LOGGING"`
	LogMode      string    `db:"name:LOG_MODE"`
	PDBs         []PdbInfo `db:"-"`
	PDBTotal     int       `db:"-"`
	PDBActive    int       `db:"-"`
}

type OracleInstance struct {
	DiscoveredSid string
	// Instance Info

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
	cmutex       sync.Mutex
	labels       map[string]string
}

func (oi *OracleInstance) String() string {
	return fmt.Sprintf("[ Discovered %s,  INSTANCE INFO: [%+v], DBInfo [%+v]", oi.DiscoveredSid, oi.InstInfo, oi.DBInfo)
}

func (oi *OracleInstance) GetExtraLabels() map[string]string {
	oi.cmutex.Lock()
	defer oi.cmutex.Unlock()
	return oi.labels
}

func (oi *OracleInstance) GetIsValidForDBQuery() bool {
	oi.cmutex.Lock()
	defer oi.cmutex.Unlock()
	return oi.IsValidForDBQuery
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
		match, err := regexp.MatchString(rule.SidRegex, SID)
		if err != nil {
			oi.log.Warnf("EXTRA LABELS: on rule[%d] matching regexp %s: error: %s", n, rule.SidRegex, err)
			return oi.labels
		}
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
	oi.cmutex.Lock()
	defer oi.cmutex.Unlock()
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

func (oi *OracleInstance) UpdateInfo() error {
	oi.cmutex.Lock()
	defer oi.cmutex.Unlock()
	// Initialize instance Data.
	// tested on 11.2.0.4.0/12.1.0.2.0/19.7.0.0.0
	log.Infof("[DISCOVERY] Initialize/Update Instance Info...")
	query := `
	select 
		INSTANCE_NUMBER,
		INSTANCE_NAME,
		HOST_NAME,
		VERSION_FULL as VERSION,
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
	if oi.ClusteWareEnabled {
		query = `SELECT MIN(INSTANCE_NUMBER) AS MIN FROM GV$INSTANCE`
		rows_db, err := oi.conn.QueryContext(ctx, query)
		if err != nil {
			log.Warnf("[DISCOVERY] Error in database Query:%s", err)
			return err
		}
		defer rows_db.Close()
		var min int
		rowsCount = 0
		for rows_db.Next() {
			err = rows_db.Scan(&min)
			if err != nil {
				return err
			}
			rowsCount += 1
		}
		log.Debugf("[DISCOVERY] Cluster Rows:%d (MIN: %d)", rowsCount, min)
		if oi.InstInfo.InstNumber == min {
			log.Debugf("This instance %s[%d] is the lowest instance => Is Valid for DB Queries ", oi.InstInfo.InstName, oi.InstInfo.InstNumber)
			oi.IsValidForDBQuery = true
		}
	}

	// Initialice PDB's info.
	log.Infof("[DISCOVERY] Initialize/Update PDB Info...")
	oi.DBInfo.PDBs = nil
	query = "select CON_ID,NAME,OPEN_MODE from v$pdbs"
	rows_pdb, err := oi.conn.QueryContext(ctx, query)
	if err != nil {
		log.Warnf("[DISCOVERY] Error in PDB Query:%s", err)
		return err
	}
	defer rows_pdb.Close()

	rowsCount = 0
	activeCount := 0
	for rows_pdb.Next() {
		pdb := PdbInfo{}
		err = rows_pdb.Scan(
			&pdb.ConID,
			&pdb.Name,
			&pdb.OpenMode,
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
		match, err := regexp.MatchString(rule.SidRegex, oi.DiscoveredSid)
		if err != nil {
			log.Warnf("ORACLE INIT: Error on rule[%d] matching regexp %s: error: %s", n, rule.SidRegex, err)
			continue
		}
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
	oi.cmutex.Lock()
	oi.conn, err = sql.Open("godror", connStr)
	if err != nil {
		log.Warnf("[DISCOVERY] Can't create connection: %s ", err)
		oi.cmutex.Unlock()
		return err
	}
	log.Tracef("[DISCOVERY] Connection String: %s", connStr)
	oi.conn.SetConnMaxLifetime(0)
	oi.conn.SetMaxIdleConns(3)
	oi.conn.SetMaxOpenConns(3)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Connection Ping
	err = oi.conn.PingContext(ctx)
	if ctx.Err() == context.DeadlineExceeded {
		oi.cmutex.Unlock()
		return errors.New("Oracle Ping timed out")
	}
	if err != nil {
		log.Warnf("[DISCOVERY] Can't ping connection: %s ", err)
		oi.cmutex.Unlock()
		return err
	}
	oi.cmutex.Unlock()
	return oi.UpdateInfo()
}

func (oi *OracleInstance) End() error {
	oi.cmutex.Lock()
	defer oi.cmutex.Unlock()
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
