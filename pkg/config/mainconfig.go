package config

// GeneralConfig has miscellaneous configuration options
type GeneralConfig struct {
	InstanceID string `mapstructure:"instanceID"`
	LogDir     string `mapstructure:"logdir"`
	HomeDir    string `mapstructure:"homedir"`
	DataDir    string `mapstructure:"datadir"`
	LogLevel   string `mapstructure:"loglevel"`
}

//SelfMonConfig configuration for self monitoring
/*type SelfMonConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	Freq              int      `mapstructure:"freq"`
	Prefix            string   `mapstructure:"prefix"`
	InheritDeviceTags bool     `mapstructure:"inheritdevicetags"`
	ExtraTags         []string `mapstructure:"extra-tags"`
}*/

// HTTPConfig has webserver config options
type HTTPConfig struct {
	BindAddr      string `mapstructure:"bind-addr"`
	AdminUser     string `mapstructure:"admin-user"`
	AdminPassword string `mapstructure:"admin-passwd"`
	CookieID      string `mapstructure:"cookie-id"`
}

// Config Main Configuration struct
type Config struct {
	General GeneralConfig
	// Database DatabaseCfg
	// Selfmon  SelfMonConfig
}

// var MainConfig Config
