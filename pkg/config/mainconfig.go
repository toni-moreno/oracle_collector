package config

import (
	"time"
)

// GeneralConfig has miscellaneous configuration options
type GeneralConfig struct {
	InstanceID string `mapstructure:"instanceID"`
	LogDir     string `mapstructure:"logdir"`
	HomeDir    string `mapstructure:"homedir"`
	DataDir    string `mapstructure:"datadir"`
	LogLevel   string `mapstructure:"loglevel"`
}

type DiscoveryConfig struct {
	OracleDiscoveryInterval time.Duration `mapstructure:"oracle_discovery_interval"`
	OracleDiscoverySidRegex string        `mapstructure:"oracle_discovery_sid_regex"`
	OracleConnectUser       string        `mapstructure:"oracle_connect_user"`
	OracleConnectPass       string        `mapstructure:"oracle_connect_pass"`
	OracleConnectDSN        string        `mapstructure:"oracle_connect_dsn"`
}

type OutputConfig struct {
	FlushPeriod time.Duration `mapstructure:"flush-period"`
	BufferSize  int           `mapstructure:"buffer-size"`
}

//SelfMonConfig configuration for self monitoring
/*type SelfMonConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	Freq              int      `mapstructure:"freq"`
	Prefix            string   `mapstructure:"prefix"`
	InheritDeviceTags bool     `mapstructure:"inheritdevicetags"`
	ExtraTags         []string `mapstructure:"extra-tags"`
}*/

type OracleMetricConfig struct {
	Context          string            `mapstructure:"context"`
	Labels           []string          `mapstructure:"labels"`
	MetricsDesc      map[string]string `mapstructure:"metricsdesc"`
	MetricsType      map[string]string `mapstructure:"metricstype"`
	FieldToAppend    string            `mapstructure:"fieldtoppend"`
	Request          string            `mapstructure:"request"`
	IgnoreZeroResult bool              `mapstructure:"ignorezeroresult"`
	// MetricsBuckets   map[string]map[string]string
}

type OracleMetricGroupConfig struct {
	QueryPeriod   time.Duration        `mapstructure:"query-period"`
	Name          string               `mapstructure:"name"`
	OracleMetrics []OracleMetricConfig `mapstructure:"MetricGroup.metric"`
}

type OracleMonitorConfig struct {
	MetricGroups []OracleMetricGroupConfig `mapstructure:"MetricGroup"`
}

// Config Main Configuration struct
type Config struct {
	General   GeneralConfig       `mapstructure:"general"`
	Output    OutputConfig        `mapstructure:"output"`
	Discovery DiscoveryConfig     `mapstructure:"oracle-discovery"`
	OraMon    OracleMonitorConfig `mapstructure:"oracle-monitor"`
	// Database DatabaseCfg
	// Selfmon  SelfMonConfig
}

// var MainConfig Config
