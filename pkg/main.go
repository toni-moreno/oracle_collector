package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/toni-moreno/oracle_collector/pkg/agent"
	"github.com/toni-moreno/oracle_collector/pkg/agent/oracle"
	"github.com/toni-moreno/oracle_collector/pkg/agent/output"
	"github.com/toni-moreno/oracle_collector/pkg/agent/selfmon"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

var (
	log  = logrus.New()
	quit = make(chan struct{})
	// startTime  = time.Now()
	getversion bool
	appdir     = os.Getenv("PWD")
	// homeDir    string
	pidFile string
	logDir  = filepath.Join(appdir, "log")
	confDir = filepath.Join(appdir, "conf")
	// dataDir    = confDir
	configFile = filepath.Join(confDir, "oracle_collector.toml")
)

func writePIDFile() {
	if pidFile == "" {
		return
	}

	// Ensure the required directory structure exists.
	err := os.MkdirAll(filepath.Dir(pidFile), 0o700)
	if err != nil {
		log.Fatal(3, "Failed to verify pid directory", err)
	}

	// Retrieve the PID and write it.
	pid := strconv.Itoa(os.Getpid())
	if err := ioutil.WriteFile(pidFile, []byte(pid), 0o644); err != nil {
		log.Fatal(3, "Failed to write pidfile", err)
	}
}

func flags() *flag.FlagSet {
	var f flag.FlagSet
	f.BoolVar(&getversion, "version", getversion, "display the version")

	//--------------------------------------------------------------
	f.StringVar(&configFile, "config", configFile, "config file")
	f.StringVar(&logDir, "logdir", logDir, "log directory  where to create al logs")
	// f.StringVar(&homeDir, "home", homeDir, "home directory")
	// f.StringVar(&dataDir, "data", dataDir, "Data directory")
	f.StringVar(&pidFile, "pidfile", pidFile, "path to pid file")
	//---------------------------------------------------------------
	f.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		f.VisitAll(func(flag *flag.Flag) {
			format := "%10s: %s\n"
			fmt.Fprintf(os.Stderr, format, "-"+flag.Name, flag.Usage)
		})
		fmt.Fprintf(os.Stderr, "\nAll settings can be set in config file: %s\n", configFile)
		os.Exit(1)
	}
	return &f
}

func init() {
	// Log format
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.Formatter = customFormatter
	customFormatter.FullTimestamp = true

	// parse first time to see if config file is being specified
	f := flags()
	f.Parse(os.Args[1:])

	if getversion {
		t, _ := strconv.ParseInt(agent.BuildStamp, 10, 64)
		fmt.Printf("oracle_collector v%s (git: %s ) built at [%s]\n", agent.Version, agent.Commit, time.Unix(t, 0).Format("2006-01-02 15:04:05"))
		os.Exit(0)
	}
	var cfg *config.Config
	// now load up config settings
	if _, err := os.Stat(configFile); err == nil {
		cfg, err = config.LoadConfigFile(configFile)
		if err != nil {
			log.Errorf("Fatal error config file: %s \n", err)
			os.Exit(1)
		}
		err = cfg.Validate()
		if err != nil {
			log.Errorf("Fatal error in config file: %s \n", err)
			os.Exit(1)
		}
		agent.MainConfig = *cfg

		// viper.SetConfigFile(configFile)
		confDir = filepath.Dir(configFile)
	} else {
		log.Errorf("Fatal error config file: %s \n", err)
		os.Exit(1)
	}

	if len(logDir) == 0 {
		logDir = cfg.General.LogDir
		// log.Infof("Set logdir %s from Command Line parameter", logDir)
	}

	// Log to file
	logfilename := logDir + "/oracle_collector.log"
	file, _ := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	// default output to console
	log.Out = file

	if len(cfg.General.LogLevel) > 0 {
		l, _ := logrus.ParseLevel(cfg.General.LogLevel)
		log.Level = l
		log.Infof("Set log level to  %s from Config File", cfg.General.LogLevel)
	}

	config.SetLogger(log)
	config.SetLogDir(logDir)
	output.SetLogger(log)
	agent.SetLogger(log)
	oracle.SetLogDir(logDir)
	oracle.SetLogger(log)
	selfmon.SetLogger(log)

	log.Debugf("Loaded Config  :%+v", cfg)
	log.Infof("Set Default directories : \n   - Exec: %s\n   - Config: %s\n   -Logs: %s\n", appdir, confDir, logDir)

	// Check Monitor Config
	cfg.OraMon.Resume(file)
}

func main() {
	defer func() {
		// errorLog.Close()
	}()
	writePIDFile()
	// Init BD config
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		select {
		case sig := <-c:
			switch sig {
			case syscall.SIGTERM:
				log.Infof("Received TERM signal")
				agent.End()
				log.Infof("Exiting for requested user SIGTERM")
				os.Exit(1)
			case syscall.SIGINT:
				log.Infof("Received INT signal")
				agent.End()
				log.Infof("Exiting for requested user SIGINT")
				os.Exit(1)
			case syscall.SIGHUP:
				log.Infof("Received HUP signal")
				agent.ReloadConf()
			}
		}
	}()

	agent.Start()

	// parse input data
}
