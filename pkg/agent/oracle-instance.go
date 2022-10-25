package agent

import (
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"

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

	AlertLogFile string                  `db:"-"`
	ListenerIP   string                  `db:"-"`
	ListenerPort int                     `db:"-"`
	PMONpid      int32                   `db:"-"`
	cfg          *config.DiscoveryConfig `db:"-"`
	conn         *go_ora.Connection      `db:"-"`
}

func (oi *OracleInstance) InitDBData() error {
	var err error

	dsn := strings.ReplaceAll(oi.cfg.OracleConnectDSN, "SID", oi.DiscoveredSid)
	connStr := "oracle://" + oi.cfg.OracleConnectUser + ":" + oi.cfg.OracleConnectPass + "@" + dsn
	oi.conn, err = go_ora.NewConnection(connStr)
	if err != nil {
		log.Warnf("Can't create connection: %s ", err)
		return err
	}
	err = oi.conn.Open()
	if err != nil {
		log.Warnf("Can't open connection: %s ", err)
		return err
	}
	// Initialize instance Data.
	// tested on 11.2.0.4.0/12.1.0.2.0/19.7.0.0.0
	log.Info("Initialize Instance Info...")
	stmt := go_ora.NewStmt("select INSTANCE_NUMBER,INSTANCE_NAME,HOST_NAME,VERSION,STARTUP_TIME,STATUS,DATABASE_STATUS,INSTANCE_ROLE,ACTIVE_STATE,BLOCKED,SHUTDOWN_PENDING from V$INSTANCE", oi.conn)
	rows, err := stmt.Query_(nil)
	if err != nil {
		log.Warnf("Error in instance Query:%s", err)
		return err
	}
	rowsCount := 0
	for rows.Next_() {
		err = rows.Scan(&(oi.InstInfo))
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
	stmt = go_ora.NewStmt("select DBID,NAME,CREATED,DB_UNIQUE_NAME,OPEN_MODE from v$database", oi.conn)
	rows, err = stmt.Query_(nil)
	if err != nil {
		log.Warnf("Error in database Query:%s", err)
		return err
	}
	rowsCount = 0
	for rows.Next_() {
		err = rows.Scan(&(oi.DBInfo))
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

func ScanSystemForInstances(procPattern string) ([]OracleInstance, error) {
	DetectedInstances := []OracleInstance{}

	pf := ProcessFinder{}

	pmonfound, err := pf.FullPattern(procPattern)
	if err != nil {
		log.Error(err)
	}

	for sid, proc := range pmonfound {

		orainst := OracleInstance{
			DiscoveredSid: sid,
			PMONpid:       proc.Pid,
		}
		DetectedInstances = append(DetectedInstances, orainst)
	}

	return DetectedInstances, nil
}
