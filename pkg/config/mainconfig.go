package config

import (
	"time"
)

// GeneralConfig has miscellaneous configuration options
type GeneralConfig struct {
	InstanceID string `toml:"instanceID"`
	LogDir     string `toml:"logdir"`
	HomeDir    string `toml:"homedir"`
	DataDir    string `toml:"datadir"`
	LogLevel   string `toml:"loglevel"`
}

type DinamicLabels struct {
	SidRegex    string            `toml:"sid_regex"`
	ExtraLabels map[string]string `toml:"extra_labels"`
}

type DiscoveryConfig struct {
	OracleDiscoveryInterval time.Duration     `toml:"oracle_discovery_interval"`
	OracleDiscoverySidRegex string            `toml:"oracle_discovery_sid_regex"`
	OracleConnectUser       string            `toml:"oracle_connect_user"`
	OracleConnectPass       string            `toml:"oracle_connect_pass"`
	OracleConnectDSN        string            `toml:"oracle_connect_dsn"`
	ExtraLabels             map[string]string `toml:"extra-labels"`
	DinamicLabelsBySID      []*DinamicLabels  `toml:"dinamic-labels"`
}

type OutputConfig struct {
	FlushPeriod time.Duration `toml:"flush_period"`
	BufferSize  int           `toml:"buffer_size"`
}

//SelfMonConfig configuration for self monitoring
/*type SelfMonConfig struct {
	Enabled           bool     `toml:"enabled"`
	Freq              int      `toml:"freq"`
	Prefix            string   `toml:"prefix"`
	InheritDeviceTags bool     `toml:"inheritdevicetags"`
	ExtraTags         []string `toml:"extra-tags"`
}*/

type OracleMetricConfig struct {
	Context          string            `toml:"context"`
	Labels           []string          `toml:"labels"`
	MetricsDesc      map[string]string `toml:"metrics_desc"`
	MetricsType      map[string]string `toml:"metrics_type"`
	FieldToAppend    string            `toml:"fieldtoppend"`
	Request          string            `toml:"request"`
	IgnoreZeroResult bool              `toml:"ignorezeroresult"`
	// MetricsBuckets   map[string]map[string]string
}

type OracleMetricGroupConfig struct {
	QueryPeriod    time.Duration        `toml:"query_period"`
	QueryTimeout   time.Duration        `toml:"query_timeout"`
	Name           string               `toml:"name"`
	InstanceFilter string               `toml:"instance_filter"`
	OracleMetrics  []OracleMetricConfig `toml:"metric"`
}

type OracleMonitorConfig struct {
	DefaultQueryTimeout time.Duration              `toml:"default_query_timeout"`
	DefaultQueryPeriod  time.Duration              `toml:"default_query_period"`
	MetricGroup         []*OracleMetricGroupConfig `toml:"mgroup"`
}

// Config Main Configuration struct
type Config struct {
	General   GeneralConfig       `toml:"general"`
	Output    OutputConfig        `toml:"output"`
	Discovery DiscoveryConfig     `toml:"oracle-discovery"`
	OraMon    OracleMonitorConfig `toml:"oracle-monitor"`
	// Database DatabaseCfg
	// Selfmon  SelfMonConfig
}

// var MainConfig Config
