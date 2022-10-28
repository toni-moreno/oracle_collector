package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "github.com/sijms/go-ora/v2"
	"github.com/sirupsen/logrus"

	"github.com/toni-moreno/oracle_collector/pkg/agent/data"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

type InstanceInfo struct {
	InstNumber      int    `db:"name:INSTANCE_NUMBER"`
	InstName        string `db:"name:INSTANCE_NAME"`
	HostName        string `db:"name:HOST_NAME"`
	Version         string `db:"name:VERSION"`
	StartupTime     string `db:"name:STARTUP_TIME"`
	Status          string `db:"name:STATUS"`
	DBStatus        string `db:"name:DATABASE_STATUS"`
	InstanceRole    string `db:"name:INSTANCE_ROLE"`
	ActiveState     string `db:"name:ACTIVE_STATE"`
	Bloqued         string `db:"name:BLOCKED"`
	ShutdownPending string `db:"name:SHUTDOWN_PENDING"`
}

type DatabaseInfo struct {
	DBID       string `db:"name:DBID"`
	DbName     string `db:"name:NAME"`
	Created    string `db:"name:CREATED"`
	DBUniqName string `db:"name:DB_UNIQUE_NAME"`
	CDB        string `db:"name:CDB"`
	OpenMode   string `db:"name:OPEN_MODE"`
}

type OracleInstance struct {
	DiscoveredSid string
	// Instance Info
	InstInfo InstanceInfo
	DBInfo   DatabaseInfo

	AlertLogFile string
	ListenerIP   string
	ListenerPort int
	PMONpid      int32
	cfg          *config.DiscoveryConfig
	conn         *sql.DB
	log          *logrus.Logger
	cmutex       sync.Mutex
}

func (oi *OracleInstance) GetExtraLabels() map[string]string {
	labels := make(map[string]string)
	// First fixed labels
	for k, v := range oi.cfg.ExtraLabels {
		labels[k] = v
	}
	// Dinamic labels.
	SID := oi.InstInfo.InstName // oi.Discovered SID

	for n, rule := range oi.cfg.DynamicLabelsBySID {
		oi.log.Debugf("Applying rule [%d] info with sid_regex = %s", n, rule.SidRegex)
		match, err := regexp.MatchString(rule.SidRegex, SID)
		if err != nil {
			oi.log.Warnf("Error on rule[%d] matching regexp %s: error: %s", n, rule.SidRegex, err)
			return labels
		}
		if match {
			for k, v := range rule.ExtraLabels {
				labels[k] = v
			}
		}
	}
	// Oracle Mandatory Labels.

	labels["instance"] = oi.InstInfo.InstName
	// labels["instance_num"] = strconv.Itoa(oi.InstInfo.InstNumber)
	labels["instance_role"] = oi.InstInfo.InstanceRole
	labels["db"] = oi.DBInfo.DbName
	// labels["db_unique_name"] = oi.DBInfo.DBUniqName

	return labels
}

func (oi *OracleInstance) Query(timeout time.Duration, query string, t *data.DataTable) (int, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	oi.cmutex.Lock()
	rows, err := oi.conn.QueryContext(ctx, query) // DATA RACE FOUND
	oi.cmutex.Unlock()
	if ctx.Err() == context.DeadlineExceeded {
		return 0, 0, errors.New("Oracle query timed out")
	}
	if err != nil {
		elapsed := time.Since(start)
		return 0, elapsed, fmt.Errorf("Error in instance Query:%s", err)
	}
	c, err := rows.Columns()
	if err != nil {
		elapsed := time.Since(start)
		return 0, elapsed, fmt.Errorf("Error on Query Columns:%s", err)
	}
	t.SetHeader(c)
	defer rows.Close()

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
	// Initialize instance Data.
	// tested on 11.2.0.4.0/12.1.0.2.0/19.7.0.0.0
	oi.Infof("[DISCOVERY] Initialize/Update Instance Info...")
	query := "select INSTANCE_NUMBER,INSTANCE_NAME,HOST_NAME,VERSION,STARTUP_TIME,STATUS,DATABASE_STATUS,INSTANCE_ROLE,ACTIVE_STATE,BLOCKED,SHUTDOWN_PENDING from V$INSTANCE"
	oi.cmutex.Lock()
	defer oi.cmutex.Unlock()

	rows, err := oi.conn.Query(query)
	if err != nil {
		oi.Warnf("Error in instance Query:%s", err)
		return err
	}
	defer rows.Close()
	rowsCount := 0
	for rows.Next() {
		err = rows.Scan(
			&oi.InstInfo.InstNumber,
			&oi.InstInfo.InstName,
			&oi.InstInfo.HostName,
			&oi.InstInfo.Version,
			&oi.InstInfo.StartupTime,
			&oi.InstInfo.Status,
			&oi.InstInfo.DBStatus,
			&oi.InstInfo.InstanceRole,
			&oi.InstInfo.ActiveState,
			&oi.InstInfo.Bloqued,
			&oi.InstInfo.ShutdownPending,
		)
		if err != nil {
			return err
		}
		rowsCount += 1
	}
	oi.Debugf("[DISCOVERY] Instance Rows:%d", rowsCount)

	// Initialize DB Data Only if instance in OPEN mode.

	if oi.InstInfo.Status != "OPEN" {
		return nil
	}
	oi.Infof("[DISCOVERY] Initialize/Update Database Info...")
	query = "select DBID,NAME,CREATED,DB_UNIQUE_NAME,OPEN_MODE from v$database"
	rows, err = oi.conn.Query(query)
	if err != nil {
		oi.Warnf("[DISCOVERY] Error in database Query:%s", err)
		return err
	}
	defer rows.Close()
	rowsCount = 0
	for rows.Next() {
		err = rows.Scan(
			&oi.DBInfo.DBID,
			&oi.DBInfo.DbName,
			&oi.DBInfo.Created,
			&oi.DBInfo.DBUniqName,
			&oi.DBInfo.OpenMode,
		)
		if err != nil {
			return err
		}
		rowsCount += 1
	}
	oi.Debugf("[DISCOVERY] DB Rows:%d", rowsCount)
	oi.Debugf("[DISCOVERY] Found %+v", oi)
	return nil
}

func (oi *OracleInstance) Init(loglevel string) error {
	var err error

	oi.log = CreateLoggerForSid(oi.DiscoveredSid, loglevel)

	dsn := strings.ReplaceAll(oi.cfg.OracleConnectDSN, "SID", oi.DiscoveredSid)
	connStr := "oracle://" + oi.cfg.OracleConnectUser + ":" + oi.cfg.OracleConnectPass + "@" + dsn
	oi.cmutex.Lock()
	oi.conn, err = sql.Open("oracle", connStr)
	if err != nil {
		oi.Warnf("Can't create connection: %s ", err)
		oi.cmutex.Unlock()
		return err
	}
	oi.conn.SetConnMaxLifetime(0)
	oi.conn.SetMaxIdleConns(3)
	oi.conn.SetMaxOpenConns(3)
	err = oi.conn.Ping()
	if err != nil {
		oi.Warnf("Can't ping connection: %s ", err)
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
		oi.Errorf("Error while closing oracle connection: %s:", err)
	}
	return nil
}
