package agent

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/toni-moreno/oracle_collector/pkg/agent/oracle"
	"github.com/toni-moreno/oracle_collector/pkg/agent/output"
	"github.com/toni-moreno/oracle_collector/pkg/agent/selfmon"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

var (
	// Version is the app X.Y.Z version
	Version string
	// Commit is the git commit sha1
	Commit string
	// Branch is the git branch
	Branch string
	// BuildStamp is the build timestamp
	BuildStamp string
)

// RInfo contains the agent's release and version information.
type RInfo struct {
	InstanceID string
	Version    string
	Commit     string
	Branch     string
	BuildStamp string
}

// GetRInfo returns the agent release information.
func GetRInfo() *RInfo {
	info := &RInfo{
		InstanceID: MainConfig.General.InstanceID,
		Version:    Version,
		Commit:     Commit,
		Branch:     Branch,
		BuildStamp: BuildStamp,
	}
	return info
}

var (

	// MainConfig contains the global configuration
	MainConfig config.Config

	log *logrus.Logger
	// reloadMutex guards the reloadProcess flag
	reloadMutex   sync.Mutex
	reloadProcess bool
	// mutex guards the runtime devices map access
	mutex    sync.RWMutex
	gatherWg sync.WaitGroup

	// processWg sync.WaitGroup
)

// SetLogger sets the current log output.
func SetLogger(l *logrus.Logger) {
	log = l
}

// End stops all devices polling.
func End() (time.Duration, error) {
	start := time.Now()
	// nothing to do
	return time.Since(start), nil
}

func Start() {
	done := make(chan bool)
	// init SelfMonitoring
	selfmon.Init(&MainConfig.Selfmon)

	// init Output Sync process
	output.Init(&MainConfig.Output)
	// init discovery process
	oracle.InitDiscovery(&MainConfig.Discovery, done)
	// init SystemMonitor Process

	cfg := MainConfig.OraMon

	chains := make([]chan bool, len(cfg.MetricGroup))
	for i, group := range cfg.MetricGroup {
		chains[i] = make(chan bool)
		log.Infof("[COLLECTOR] Begin [%d] Collecting data from Group [%s]", i, group.Name)
		processor := InitGroupProcessor(group, oracle.OraList)
		processor.StartCollection(chains[i], &gatherWg)
	}
	// init OracleMonitor Process
	gatherWg.Wait()
}

// ReloadConf stops the polling, reloads all configuration and restart the polling.
func ReloadConf() (time.Duration, error) {
	start := time.Now()
	// nothing to do yet
	return time.Since(start), nil
}
