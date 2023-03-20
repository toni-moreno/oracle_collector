package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
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

func (gc *GeneralConfig) Validate() error {
	return nil
}

type DinamicParams struct {
	SidRegex          string            `toml:"sid_regex"`
	R                 *regexp.Regexp    `toml:"-"`
	ExtraLabels       map[string]string `toml:"extra_labels"`
	OracleConnectUser string            `toml:"oracle_connect_user"`
	OracleConnectPass string            `toml:"oracle_connect_pass"`
	OracleConnectDSN  string            `toml:"oracle_connect_dsn"`
}

func (dp *DinamicParams) Validate() error {
	r, err := regexp.Compile(dp.SidRegex)
	if err != nil {
		return fmt.Errorf("Error on Dinamic Params: %s: %s", dp.SidRegex, err)
	}
	dp.R = r
	return nil
}

type DiscoveryConfig struct {
	OracleClusterwareEnabled       bool              `toml:"oracle_clusterware_enabled"`
	OracleDiscoveryInterval        time.Duration     `toml:"oracle_discovery_interval"`
	OracleDiscoverySidRegex        string            `toml:"oracle_discovery_sid_regex"`
	OracleDiscoverySkipErrorsRegex []string          `toml:"oracle_discovery_skip_errors_regex"`
	SkipErrR                       []*regexp.Regexp  `toml:"-"`
	OracleConnectUser              string            `toml:"oracle_connect_user"`
	OracleConnectPass              string            `toml:"oracle_connect_pass"`
	OracleConnectDSN               string            `toml:"oracle_connect_dsn"`
	ExtraLabels                    map[string]string `toml:"extra_labels"`
	OracleStatusExtendedInfo       bool              `toml:"oracle_status_extended_info"`
	OracleLogLevel                 string            `toml:"oracle_log_level"`
	DynamicParamsBySID             []*DinamicParams  `toml:"dynamic-params"`
}

func (dc *DiscoveryConfig) Validate() error {
	if len(dc.OracleConnectDSN) == 0 {
		return fmt.Errorf("Discovery Config  parameter: oracle_connect_dsn is mandatory")
	}
	if len(dc.OracleConnectUser) == 0 {
		return fmt.Errorf("Discovery Config  parameter: oracle_connect_user is mandatory")
	}
	if len(dc.OracleConnectPass) == 0 {
		return fmt.Errorf("Discovery Config  parameter: oracle_connect_pass is mandatory")
	}
	_, err := regexp.Compile(dc.OracleDiscoverySidRegex)
	if err != nil {
		return fmt.Errorf("Error on Discovery Config  parameter  oracle_discovery_sid_regex : %s", err)
	}

	for _, rexp := range dc.OracleDiscoverySkipErrorsRegex {
		r, err := regexp.Compile(rexp)
		if err != nil {
			return fmt.Errorf("Error on Skip Error Param regex: %s: %s", rexp, err)
		}
		dc.SkipErrR = append(dc.SkipErrR, r)
	}

	for _, v := range dc.DynamicParamsBySID {
		err := v.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

type OutputConfig struct {
	FlushPeriod time.Duration `toml:"flush_period"`
	BufferSize  int           `toml:"buffer_size"`
	BatchSize   int           `toml:"batch_size"`
}

func (oc *OutputConfig) Validate() error {
	return nil
}

// SelfMonConfig configuration for self monitoring
type SelfMonConfig struct {
	Enabled     bool              `toml:"enabled"`
	ReqPeriod   time.Duration     `toml:"request_period"`
	Prefix      string            `toml:"measurement_prefix"`
	ExtraLabels map[string]string `toml:"extra_labels"`
}

func (sc *SelfMonConfig) Validate() error {
	return nil
}

// InheritDeviceTags bool          `toml:"inherit-intance-labels"`
type OracleMetricConfig struct {
	ID                       string            `toml:"id"`
	OraVerGreaterOrEqualThan string            `toml:"oracle_version_greater_or_equal_than"`
	OraVerLessThan           string            `toml:"oracle_version_less_than"`
	Context                  string            `toml:"context"`
	Labels                   []string          `toml:"labels"`
	MetricsDesc              map[string]string `toml:"metrics_desc"`
	MetricsType              map[string]string `toml:"metrics_type"`
	FieldToAppend            string            `toml:"fieldtoappend"`
	Request                  string            `toml:"request"`
	IgnoreZeroResult         bool              `toml:"ignorezeroresult"`
	// MetricsBuckets   map[string]map[string]string
}

func (mc *OracleMetricConfig) Validate() error {
	if len(mc.Context) == 0 {
		return fmt.Errorf("Metric Config context  parameter is mandatory")
	}
	if len(mc.Request) == 0 {
		return fmt.Errorf("Metric Config request  parameter is mandatory")
	}
	if len(mc.MetricsType) == 0 {
		return fmt.Errorf("Metric Config metric_type parameter is mandatory")
	}
	if len(mc.ID) == 0 {
		mc.ID = mc.Context
	}

	for k, v := range mc.MetricsType {
		switch v {
		case "INTEGER", "COUNTER", "integer", "counter", "float", "FLOAT", "bool", "BOOL", "BOOLEAN", "string", "STRING":
		default:
			return fmt.Errorf("Error in Metric %s , Type error in field %s:  Valid types are [INTEGER,COUNTER,integer,counter,float,FLOAT,bool,BOOL,BOOLEAN,string,STRING", mc.ID, k)
		}
	}
	return nil
}

type OracleMetricGroupConfig struct {
	QueryLevel     string                `toml:"query_level"` // db/instance default  instance
	QueryPeriod    time.Duration         `toml:"query_period"`
	QueryTimeout   time.Duration         `toml:"query_timeout"`
	Name           string                `toml:"name"`
	InstanceFilter string                `toml:"instance_filter"`
	OracleMetrics  []*OracleMetricConfig `toml:"metric"`
}

func (gc *OracleMetricGroupConfig) Validate() error {
	if len(gc.Name) == 0 {
		return fmt.Errorf("Metric Group Config name  parameter is mandatory")
	}
	// set default value
	if len(gc.QueryLevel) == 0 {
		gc.QueryLevel = "instance"
	}

	for _, v := range gc.OracleMetrics {
		err := v.Validate()
		if err != nil {
			return fmt.Errorf("Error in MetricGroup %s : %s", gc.Name, err)
		}
	}
	return nil
}

/*func (omgc *OracleMetricGroupConfig) GetQueryLevel() string {
	if len(omgc.QueryLevel) > 0 {
		return omgc.QueryLevel
	}
	return "instance"
}*/

type OracleMonitorConfig struct {
	DefaultQueryTimeout time.Duration              `toml:"default_query_timeout"`
	DefaultQueryPeriod  time.Duration              `toml:"default_query_period"`
	MetricGroup         []*OracleMetricGroupConfig `toml:"mgroup"`
}

func (om *OracleMonitorConfig) Validate() error {
	for _, v := range om.MetricGroup {
		err := v.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (om *OracleMonitorConfig) Resume(f *os.File) {
	w := bufio.NewWriter(f)
	w.WriteString("**==========================================================================================\n")
	for _, mgc := range om.MetricGroup {
		s := fmt.Sprintf("** GROUP: %s [Level:%s] [Period:%s|Timeout:%s ]\n",
			mgc.Name,
			mgc.QueryLevel,
			mgc.QueryPeriod,
			mgc.QueryTimeout)
		w.WriteString(s)
		for _, mc := range mgc.OracleMetrics {
			s := fmt.Sprintf("**\t\t\t METRIC ID: %s | CONTEXT: %s | Version [%s,%s)[%d labels|%d fields]\n",
				mc.ID,
				mc.Context,
				mc.OraVerGreaterOrEqualThan,
				mc.OraVerLessThan,
				len(mc.Labels),
				len(mc.MetricsType))
			w.WriteString(s)
		}
	}
	w.WriteString("**==========================================================================================\n")
	w.Flush()
}

// Config Main Configuration struct
type Config struct {
	General   *GeneralConfig       `toml:"general"`
	Output    *OutputConfig        `toml:"output"`
	Selfmon   *SelfMonConfig       `toml:"self-monitor"`
	Discovery *DiscoveryConfig     `toml:"oracle-discovery"`
	OraMon    *OracleMonitorConfig `toml:"oracle-monitor"`
	// Database DatabaseCfg
}

func (c *Config) Validate() error {
	var err error
	err = c.General.Validate()
	if err != nil {
		return err
	}
	err = c.Output.Validate()
	if err != nil {
		return err
	}

	err = c.Selfmon.Validate()
	if err != nil {
		return err
	}

	err = c.OraMon.Validate()
	if err != nil {
		return err
	}

	err = c.Discovery.Validate()
	if err != nil {
		return err
	}

	return nil
}

// var MainConfig Config
