# Oracle Collector

Oracle Collector is an Open Source tool to get oracle metrics from any oracle instance running in any compatible ( Linux ) server, it runs as an execd plugin for telegraf


## Install from precompiled packages

All releases here.

[releases](https://github.com/toni-moreno/oracle_collector/releases)

## Building and Run from master

If you want to build a package yourself, or contribute. Here is a guide for how to do that.

### Dependencies

- Go 1.19

### Get Code

```bash
go get -d github.com/toni-moreno/oracle_collector/...
```

### Building the backend


```bash
cd $GOPATH/src/github.com/toni-moreno/oracle_collector
go run build.go build           
```

### Creating minimal package tar.gz

After building frontend and backend you will do

```bash
go run build.go pkg-min-tar
```

### Creating rpm and deb packages

you  will need previously installed the fpm/rpm and deb packaging tools.
After building frontend and backend  you will do.

```bash
go run build.go latest
```

### Creating  and running docker image


```bash
make -f Makefile.docker
docker run tonimoreno/oracle_collector:latest -version
docker run  tonimoreno/oracle_collector:latest -h
docker run  -p 4090:4090 -v /mylocal/conf:/opt/oracle_collector/conf -v /mylocal/log:/opt/oracle_collector/log tonimoreno/oracle_collector:latest [options]
```


### Recompile backend on source change (only for developers)

To rebuild on source change (requires that you executed godep restore)
```bash
go install github.com/unknwon/bra@latest
bra run  
```

## Running first time ( outside telegraf )

You will need to set up oracle client environment variables `LD_LIBRARY_PATH` and `ORACLE_HOME` to run the collector.

```bash
export LD_LIBRARY_PATH=/opt/oracle/product/21c/dbhomeXE/lib/
export ORACLE_HOME=/opt/oracle/product/21c/dbhomeXE
```

### Create a connection user.

You will need a monitoring user with proper grants to query all needed info.

`sqlplus "/ as sysdba" @./conf/recreate_user_C##MONIT.sql`



To execute without any configuration you need a minimal oracle_collector.toml file on the conf directory.

```bash
cp conf/sample.oracle_collector.toml conf/oracle_collector.toml
./bin/oracle_collector [options]
```

## Running as Telegraf plugin.

Oracle collector will run as telegraf execd plugin you can use the sample in the conf dir. Telegraf will be executed as root user, so you will need to setup oracle client environment variables in the execd config file.

```bash
cp conf/telegraf-execd-example.conf /etc/telegraf.d/oracle_collector.conf
systemctl restart telegraf.service
```


## Basic Usage

```bash
$ ./bin/oracle_collector -h
Usage of ./bin/oracle_collector:
   -config: config file
   -logdir: log directory where to create all log files
  -pidfile: path to pid file 
  -version: display the version
```

