# Oracle Collector

Oracle Collector is an Open Source tool to get oracle metrics from any oracle instance in 


## Install from precompiled packages

All releases here.

[releases](https://github.com/toni-moreno/oracle_collector/releases)

## Run from master

If you want to build a package yourself, or contribute. Here is a guide for how to do that.

### Dependencies

- Go 1.17

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

### Running first time
To execute without any configuration you need a minimal config.toml file on the conf directory.

```bash
cp conf/sample.oracle_collector.toml conf/oracle_collector.toml
./bin/oracle_collector [options]
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


## Basic Usage

### Execution parameters

