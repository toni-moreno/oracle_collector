package config

import (
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
)

var (
	// Log the Logger
	log     *logrus.Logger
	dataDir string
	logDir  string
	confDir string
)

// SetDirs set default dirs to set db and logs
func SetDirs(data string, log string, conf string) {
	dataDir = data
	logDir = log
	confDir = conf
}

// SetLogDir set default dirs to set db and logs
func SetLogDir(log string) {
	logDir = log
}

// SetLogger set the output log
func SetLogger(l *logrus.Logger) {
	log = l
}

func LoadConfigFile(filename string) (*Config, error) {
	cfg := &Config{}

	f, err := os.Open(filename)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	tomlData, err := ioutil.ReadAll(f)
	if err != nil {
		return cfg, err
	}

	if _, err := toml.Decode(string(tomlData), cfg); err != nil {
		return cfg, err
	}
	// // Validate Cluster Config
	// for _, c := range cfg.Influxcluster {
	// 	err := c.ValidateCfg(cfg)
	// 	if err != nil {
	// 		return cfg, err
	// 	}
	// }
	// // Validate Backend Config
	// for _, b := range cfg.Influxdb {
	// 	err := b.ValidateCfg(cfg)
	// 	if err != nil {
	// 		return cfg, err
	// 	}
	// }

	// // Validate Backend Config
	// for _, h := range cfg.HTTPConfig {
	// 	err := h.ValidateCfg(cfg)
	// 	if err != nil {
	// 		return cfg, err
	// 	}
	// }

	return cfg, err
}
