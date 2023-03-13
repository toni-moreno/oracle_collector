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

Use this for Oracle >= 12.1

`sqlplus "/ as sysdba" @./conf/recreate_user_C##MONIT.sql`

Use this for Oracle < 12.1

`sqlplus "/ as sysdba" @./conf/recreate_user_C##MONIT_legacy.sql`


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


## Gathered Info.

Oracle collector gathers 2 kind of different informations.

1. Oracle Monitored/Discovered instances 
2. Internat stats.


## Oracle Monitored Measurements/Metrics

The agent is capable to get metrics organized and sent as ILP ( InfluDB Line Protocol) measurements. It takes 2 non configurable measurements. And a lot of other configurable measurements in the `metric` section of the config file`[[oracle-monitor.mgroup.metric]]`.

### Non configurable measurements.

These measurements will be sent always even if any metric `[[oracle-monitor.mgroup.metric]]` and any metric group `[[oracle-monitor.mgroup]]` exiting in the config file.

**oracle_status**

Get info from [v$instance](https://docs.oracle.com/en/database/oracle/oracle-database/19/refrn/V-INSTANCE.html) and [v$database] and [v$pdbs]


```sql
  select
    INSTANCE_NUMBER,
    INSTANCE_NAME,
    HOST_NAME,
    -- for version <v18c (12.2.0.2)
    VERSION
    -- only for verion > v18c (12.2.0.2)
    VERSION_FULL AS VERSION,
    STARTUP_TIME,
    FLOOR((SYSDATE - STARTUP_TIME) * 60 * 60 * 24) as UPTIME,
    STATUS,
    DATABASE_STATUS,
    INSTANCE_ROLE,
    ACTIVE_STATE,
    BLOCKED,
    SHUTDOWN_PENDING,
    ARCHIVER
  from V$INSTANCE
```
And

```sql
  select
    DBID,
    NAME,
    CREATED,
    DB_UNIQUE_NAME,
    OPEN_MODE,
    DATABASE_ROLE,
    FORCE_LOGGING,
    LOG_MODE
  from v$database`
```



* **tags**
  * all `extra_labels` from the `[oracle-discovery]` config section
  * all `[global_tags]` configured in the parent telegraf config
  * *db:* name o the DB
  * *db_unique_name:* de unique name for the DB
  * *instance:* name of the instance
  * *instance_role:* Indicates whether the instance is an active instance or an inactive secondary instance.

How much info it sends to the backend depens on the `oracle_status_extended_info` flag on the  `[oracle-discovery]` section:

* **fields**
  * From system process
    * *proc_ok (boolean)*:  True when process (proc_pid) is ok, false first time when detected is down.
    * *proc_pid (integer)*: PID from the Discovered PMON process
  * From `v$instance` view:
    * *inst_number (integer)*:
    * *inst_status (string)*:
    * *inst_uptime_sec (integer)*: 
    * *inst_version (string)*:
    * *inst_active_state (string)*: (extended only)
    * *inst_startup_time: (string)*: (extended only)
    * *inst_bloqued (string)*: (extended only)
    * *inst_db_status (status)*: (extended only)
    * *inst_shutdown_pending (string)*: (extended only)
    * *inst_archiver (string)*: (extended only)
  * from `v$database` view:
    * *db_role (string)*:
    * *db_open_mode (string)*:
    * *db_created (string)*: (extended only)
    * *db_log_mode (string)*: (extended only)
    * *db_force_logging (string)*: (extended only)
  * from `v$pdbs` view:
    * *pdbs_total (integer)*:
    * *pdbs_active (integer)*:

**oracle_pdb_status**

Get info from [v$pdbs](https://docs.oracle.com/database/121/REFRN/GUID-A399F608-36C8-4DF0-9A13-CEE25637653E.htm#REFRN30652) for Oracle version > 12.1 each "discovery period" with the following query.

for Oracle < 12.1.0.2

```sql 
select 
  CON_ID,NAME,OPEN_MODE,RESTRICTED,TOTAL_SIZE,
from v$pdbs
```

for Oracle >= 12.1.0.2

```sql
select 
  CON_ID,NAME,OPEN_MODE,RESTRICTED,RECOVERY_STATUS,TOTAL_SIZE,BLOCK_SIZE
from v$pdbs
```

* **tags**
  * all `extra_labels` from the `[oracle-discovery]` config section
  * all `[global_tags]` configured in the parent telegraf config
  * *db:* name o the DB
  * *db_unique_name:* de unique name for the DB
  * *instance:* name of the instance
  * *instance_role:* Indicates whether the instance is an active instance or an inactive secondary instance.
  * *pdb_name:* the name of the pdb

* **fields**
  * *open_mode (string)*
  * *restricted (string)*
  * *recovery_status (string)*
  * *total_size (integer)*
  * *block_size (integer)*

## Configurable measurements.

On each `[oracle-monitor.mgroup.metric]` section you can define measurment name,tags,and fiends as follows. 

```toml
[[oracle-monitor.mgroup.metric]]
# Resource
id = "resource_query_XXXX"
context = "resource"
labels = [ "resource_name" ]
metrics_desc = { current_utilization= "Generic counter metric from v$resource_limit view in Oracle (current value).", limit_value="Generic counter metric from v$resource_limit view in Oracle (UNLIMITED: -1)." }
metrics_type = { current_utilization='integer',limit_value='integer',used_pct='float'}
request='''
SELECT 
    resource_name,
    current_utilization,
    CASE WHEN TRIM(limit_value) LIKE 'UNLIMITED' THEN '-1' ELSE TRIM(limit_value) END as limit_value,
    CASE 
        WHEN TRIM(limit_value) LIKE 'UNLIMITED' THEN 0 
        WHEN TRIM(limit_value) LIKE '0' THEN 0
        ELSE ROUND(((current_utilization*100)/limit_value),3)
    END as USED_PCT
FROM v$resource_limit
'''
```

- **measurement_name:** will be set with the value in the `context` struct field
- **tags:**  will be set with all common tags ,  `extra_labels` from the `[oracle-discovery]` config section and  `[global_tags]` configured in the parent telegraf config appended by the `labels` struct list. ( each label should be a column in the response query)
- **fields:** will be set and type trasnformed if needed by the definition in the in `metrics_type` struct field 

Tags and fields will taken from the resulted query in field `request`

In the above example:

**measurement:** "resource"
* **tags**
  * all `extra_labels` from the `[oracle-discovery]` config section
  * all `[global_tags]` configured in the parent telegraf config
  * resource_name
* **fields**
  * current_utilization (integer):
  * limit_value (integer)
  * used_pct(float)


## Internal Statistics.

Oracle collector gathers also some internal processes informattion. The name of its measurements depens on the `measurement_prefix` parameter from the `[self-monitor]` config section.

These are the collected measurements:

**<prefix>runtime_gvm_stats**

Gathers information about the Go Virtual Machine runtime stats. ( Garbage collection, goroutines and memory) Refer to [MemStats](https://pkg.go.dev/runtime#MemStats) to get detailed info on memory stats.

* **tags**
  * all `extra_labels` from the `[self-monitor]` config
  * all `[global_tags]` configured in the parent telegraf config
* **fields**
  * *runtime_goroutines*: returns the number of goroutines that currently exist.
  * *mem.mallocs:* is the cumulative count of heap objects allocated. 
  * *mem.alloc:*  is bytes of allocated heap objects.
  * *mem.frees:*  is the cumulative count of heap objects freed.
  * *mem.sys:* is the total bytes of memory obtained from the OS.
  * *mem.heap_alloc_bytes:*  is bytes of allocated heap objects.
  * *mem.heap_sys_bytes:* HeapSys is bytes of heap memory obtained from the OS.
  * *mem.heap_idle_bytes:* HeapIdle is bytes in idle (unused) spans.
  * *mem.heap_in_use_bytes:* HeapInuse is bytes in in-use spans.
  * *mem.heap_released_bytes:* HeapReleased is bytes of physical memory returned to the OS.
  * *mem.heap_objects:* HeapObjects is the number of allocated heap objects.
  * *mem.stack_in_use_bytes:* is bytes in stack spans.
  * *mem.m_span_in_use_bytes:* MSpanInuse is bytes of allocated mspan structures.
  * *mem.m_cache_in_use_bytes:*  MCacheInuse is bytes of allocated mcache structures.
  * *gc.total_pause_ns:* is the cumulative nanoseconds in GC stop-the-world pauses since the program started.
  * *gc.pause_per_interval:* pause on each gathering `request_period` interval.
  * *gc.pause_per_second:* pause acummulated per second.
  * *gc.gc_per_inteval:*  number of gc's per `request_period`interval.
  * *gc.gc_per_second:*  number if gc's per second.


**<prefix>discover_stats**

Gathers information about the oracle instance discovery process.

* **tags**
  * all `extra_labels` from the `[self-monitor]` config
  * all `[global_tags]` configured in the parent telegraf config
* **fields**
  * *all:* number of discovered processes with the `oracle_discovery_sid_regex` process pattern.
  * *new:* number of new instances since last discovery process.
  * *current:" number of currently discovered and connected oracle instances
  * *disconnected:" number of instances which has been disconnected since the last discovery process
  * *connect_errors*: number of oracle instances with errors in the connecting process.
  * *disconnected_sid_names: list of SID names which has beed disconnected since the last discovery process.  ( separeted by ":")
  * *connected_sid_names* list of SID names currently connected (separated by ":).
  * *errconnect_sid_names* list of SID names with connetion errors (separated by ":")


**<prefix>collect_stats**

This measurement is needed to know how long collector takes on each query on each database and how many results (metrics) are we gathering in each query.

* **tags**
  * all `extra_labels` from the `[self-monitor]` config
  * all `[global_tags]` configured in the parent telegraf config
  * *db:* the database where collecting metrics
  * *db_unique_name:* the unique name for the database
  * *instance:* the instance where connected
  * *instance_role:* the role of the instance.
  * *metric_group:* the name of the metric group where is configured 
  * *metric_context:* the context of the metric group where is configured ( `context` parameter)
  * *metric_id:* the uniq id of the metric ( `id` parameter or `context`if id not configured)
* **fields**
  * *num_metrics*: num of collected metrics from this query metric.
  * *duration_us*: duration of the query in microseconds  


**<prefix>sql_driver_stats**

Gather information on each collector to  each DB instance connection with these [sql generic stats](https://pkg.go.dev/database/sql#DBStats)

* **tags**
  * all `extra_labels` from the `[self-monitor]` config
  * all `[global_tags]` configured in the parent telegraf config
  * *instance*: the instance name 
* **fields**
  * *idle_conn*: The number of idle connections.
  * *inuse_conn*: The number of connections currently in use.
  * *max_idle_closed*: The total number of connections closed due to SetMaxIdleConns.
  * *max_idle_time_closed*: The total number of connections closed due to SetConnMaxIdleTime.
  * *max_open_connections*:
  * *open_connections*: The number of established connections both in use and idle.
  * *wait_count*: The total number of connections waited for.
  * *wait_duration_ms*: The total time blocked waiting for a new connection.