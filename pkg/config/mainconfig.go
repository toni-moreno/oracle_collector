package config

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// GeneralConfig has miscellaneous configuration options
type GeneralConfig struct {
	InstanceID string `toml:"instanceID"`
	LogDir     string `toml:"log_dir"`
	HomeDir    string `toml:"home_dir"`
	DataDir    string `toml:"data_dir"`
	LogLevel   string `toml:"log_level"`
}

type DinamicParams struct {
	SidRegex          string            `toml:"sid_regex"`
	ExtraLabels       map[string]string `toml:"extra_labels"`
	OracleConnectUser string            `toml:"oracle_connect_user"`
	OracleConnectPass string            `toml:"oracle_connect_pass"`
	OracleConnectDSN  string            `toml:"oracle_connect_dsn"`
}

type DiscoveryConfig struct {
	OracleClusterwareEnabled bool              `toml:"oracle_clusterware_enabled"`
	OracleDiscoveryInterval  time.Duration     `toml:"oracle_discovery_interval"`
	OracleDiscoverySidRegex  string            `toml:"oracle_discovery_sid_regex"`
	OracleConnectUser        string            `toml:"oracle_connect_user"`
	OracleConnectPass        string            `toml:"oracle_connect_pass"`
	OracleConnectDSN         string            `toml:"oracle_connect_dsn"`
	ExtraLabels              map[string]string `toml:"extra_labels"`
	OracleLogLevel           string            `toml:"oracle_log_level"`
	DynamicParamsBySID       []*DinamicParams  `toml:"dynamic-params"`
}

type OutputConfig struct {
	FlushPeriod time.Duration `toml:"flush_period"`
	BufferSize  int           `toml:"buffer_size"`
	BatchSize   int           `toml:"batch_size"`
}

// SelfMonConfig configuration for self monitoring
type SelfMonConfig struct {
	Enabled     bool              `toml:"enabled"`
	ReqPeriod   time.Duration     `toml:"request_period"`
	Prefix      string            `toml:"measurement_prefix"`
	ExtraLabels map[string]string `toml:"extra_labels"`
}

// InheritDeviceTags bool          `toml:"inherit-intance-labels"`
type OracleMetricConfig struct {
	Context          string            `toml:"context"`
	Labels           []string          `toml:"labels"`
	MetricsDesc      map[string]string `toml:"metrics_desc"`
	MetricsType      map[string]string `toml:"metrics_type"`
	FieldToAppend    string            `toml:"fieldtoappend"`
	Request          string            `toml:"request"`
	IgnoreZeroResult bool              `toml:"ignorezeroresult"`
	// MetricsBuckets   map[string]map[string]string
}

type QueyType uint

type OracleMetricGroupConfig struct {
	QueryLevel     string               `toml:"query_level"` // db/inst
	QueryPeriod    time.Duration        `toml:"query_period"`
	QueryTimeout   time.Duration        `toml:"query_timeout"`
	Name           string               `toml:"name"`
	InstanceFilter string               `toml:"instance_filter"`
	OracleMetrics  []OracleMetricConfig `toml:"metric"`
}

func (omgc *OracleMetricGroupConfig) GetQueryLevel() string {
	if len(omgc.QueryLevel) > 0 {
		return omgc.QueryLevel
	}
	return "instance"
}

type OracleMonitorConfig struct {
	DefaultQueryTimeout time.Duration              `toml:"default_query_timeout"`
	DefaultQueryPeriod  time.Duration              `toml:"default_query_period"`
	MetricGroup         []*OracleMetricGroupConfig `toml:"mgroup"`
}

func (om *OracleMonitorConfig) Resume(f *os.File) {
	w := bufio.NewWriter(f)
	w.WriteString("**==========================================================================================\n")
	for _, mgc := range om.MetricGroup {
		for _, mc := range mgc.OracleMetrics {
			s := fmt.Sprintf("** GROUP: %s [Level:%s] [Period:%s|Timeout:%s ]  METRIC_CONTEXT: %s\n", mgc.Name, mgc.GetQueryLevel(), mgc.QueryPeriod, mgc.QueryTimeout, mc.Context)
			w.WriteString(s)
		}
	}
	w.WriteString("**==========================================================================================\n")
	w.Flush()
}

// Config Main Configuration struct
type Config struct {
	General   GeneralConfig       `toml:"general"`
	Output    OutputConfig        `toml:"output"`
	Selfmon   SelfMonConfig       `toml:"self-monitor"`
	Discovery DiscoveryConfig     `toml:"oracle-discovery"`
	OraMon    OracleMonitorConfig `toml:"oracle-monitor"`
	// Database DatabaseCfg
}

// var MainConfig Config
