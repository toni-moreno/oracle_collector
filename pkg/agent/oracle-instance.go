package agent

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"time"

	_ "github.com/sijms/go-ora/v2"

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
	// conn         *go_ora.Connection
}

var (
	OraInstances []*OracleInstance
	OraInstMutex sync.Mutex
	processWg    sync.WaitGroup
)

func (oi *OracleInstance) Query(timeout time.Duration, query string, t *DataTable) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rows, err := oi.conn.QueryContext(ctx, query, nil)
	if ctx.Err() == context.DeadlineExceeded {
		return 0, errors.New("Oracle query timed out")
	}
	if err != nil {
		log.Warnf("Error in instance Query:%s", err)
		return 0, err
	}
	c, err := rows.Columns()
	if err != nil {
		log.Warnf("Error Query Columns:%s", err)
		return 0, err
	}
	t.SetHeader(c)
	defer rows.Close()

	for rows.Next() {
		rowpointers := t.AppendEmptyRow()
		if err := rows.Scan(rowpointers...); err != nil {
			return 0, err
		}
	}
	return t.Length(), nil
}

func (oi *OracleInstance) InitDBData() error {
	var err error

	dsn := strings.ReplaceAll(oi.cfg.OracleConnectDSN, "SID", oi.DiscoveredSid)
	connStr := "oracle://" + oi.cfg.OracleConnectUser + ":" + oi.cfg.OracleConnectPass + "@" + dsn
	oi.conn, err = sql.Open("oracle", connStr)
	if err != nil {
		log.Warnf("Can't create connection: %s ", err)
		return err
	}
	oi.conn.SetConnMaxLifetime(0)
	oi.conn.SetMaxIdleConns(3)
	oi.conn.SetMaxOpenConns(3)
	err = oi.conn.Ping()
	if err != nil {
		log.Warnf("Can't ping connection: %s ", err)
		return err
	}

	// Initialize instance Data.
	// tested on 11.2.0.4.0/12.1.0.2.0/19.7.0.0.0
	log.Info("Initialize Instance Info...")
	query := "select INSTANCE_NUMBER,INSTANCE_NAME,HOST_NAME,VERSION,STARTUP_TIME,STATUS,DATABASE_STATUS,INSTANCE_ROLE,ACTIVE_STATE,BLOCKED,SHUTDOWN_PENDING from V$INSTANCE"
	rows, err := oi.conn.Query(query)
	if err != nil {
		log.Warnf("Error in instance Query:%s", err)
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
	log.Infof("Rows count:%s", rowsCount)

	// Initialize DB Data Only if instance in OPEN mode.

	if oi.InstInfo.Status != "OPEN" {
		return nil
	}
	log.Info("Initialize Database Info...")
	query = "select DBID,NAME,CREATED,DB_UNIQUE_NAME,OPEN_MODE from v$database"
	rows, err = oi.conn.Query(query)
	if err != nil {
		log.Warnf("Error in database Query:%s", err)
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
	log.Infof("Rows count: %s ", rowsCount)

	log.Infof("Found %+v", oi)
	return nil
}

func discover(cfg *config.DiscoveryConfig) {
	oinstances, err := ScanSystemForInstances(cfg.OracleDiscoverySidRegex)
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
	OraInstMutex.Lock()
	OraInstances = oinstances
	OraInstMutex.Unlock()
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

func ScanSystemForInstances(procPattern string) ([]*OracleInstance, error) {
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
		}
		DetectedInstances = append(DetectedInstances, orainst)
	}

	return DetectedInstances, nil
}
