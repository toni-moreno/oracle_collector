-- needed to set non standard names
alter session set "_ORACLE_SCRIPT"=true;
-- Drop user if already exist
-- https://www.oracletutorial.com/oracle-administration/oracle-drop-user/
DROP USER C##MONIT CASCADE;
-- Drop profile if already exist
-- https://www.oracletutorial.com/oracle-administration/oracle-drop-profile/
DROP PROFILE "C##PROFILE_MONIT";
-- Crear profile en CDB
CREATE PROFILE "C##PROFILE_MONIT" LIMIT COMPOSITE_LIMIT UNLIMITED SESSIONS_PER_USER UNLIMITED CPU_PER_SESSION UNLIMITED CPU_PER_CALL UNLIMITED LOGICAL_READS_PER_SESSION UNLIMITED LOGICAL_READS_PER_CALL UNLIMITED IDLE_TIME UNLIMITED CONNECT_TIME UNLIMITED PRIVATE_SGA UNLIMITED FAILED_LOGIN_ATTEMPTS 10 PASSWORD_LIFE_TIME UNLIMITED PASSWORD_REUSE_TIME UNLIMITED PASSWORD_REUSE_MAX UNLIMITED PASSWORD_VERIFY_FUNCTION NULL PASSWORD_LOCK_TIME UNLIMITED PASSWORD_GRACE_TIME UNLIMITED CONTAINER=ALL;
-- Crear el usuario
CREATE USER C##MONIT IDENTIFIED BY "XXYYY" PROFILE C##PROFILE_MONIT CONTAINER=ALL;
ALTER USER C##MONIT SET CONTAINER_DATA=ALL CONTAINER=CURRENT;
ALTER USER C##MONIT QUOTA UNLIMITED ON USERS;
GRANT ALTER SESSION TO C##MONIT;
GRANT CREATE SESSION TO C##MONIT CONTAINER=ALL;
GRANT SELECT_CATALOG_ROLE TO C##MONIT;
GRANT CONNECT TO C##MONIT;
GRANT SELECT ON V_$DATABASE TO C##MONIT;
GRANT SELECT ON V_$INSTANCE TO C##MONIT;
GRANT SELECT ON GV_$INSTANCE TO C##MONIT;
GRANT SELECT ON GV_$CELL_STATE TO C##MONIT;
GRANT SELECT ON V_$VERSION TO C##MONIT;
GRANT SELECT ON GV_$VERSION TO C##MONIT;
GRANT SELECT ON V_$LOG TO C##MONIT;
GRANT SELECT ON GV_$LOG TO C##MONIT;
GRANT SELECT ON V_$LOGFILE TO C##MONIT;
GRANT SELECT ON GV_$LOGFILE TO C##MONIT;
GRANT SELECT ON V_$LOG_HISTORY TO C##MONIT;
GRANT SELECT ON GV_$LOG_HISTORY TO C##MONIT;
GRANT SELECT ON DBA_HIST_SNAPSHOT TO C##MONIT;
GRANT SELECT ON DBA_HIST_TBSPC_SPACE_USAGE TO C##MONIT;
GRANT SELECT ON V_$RECOVERY_FILE_DEST TO C##MONIT;
GRANT SELECT ON V_$RESTORE_POINT TO C##MONIT;
GRANT SELECT ON GV_$RESTORE_POINT TO C##MONIT;
GRANT SELECT ON GV_$DATAGUARD_STATS TO C##MONIT;
GRANT SELECT ON GV_$ARCHIVE_DEST_STATUS TO C##MONIT;
GRANT SELECT ON V_$BACKUP_SET_DETAILS TO C##MONIT;
GRANT SELECT ON V_$RECOVERY_AREA_USAGE TO C##MONIT;
GRANT SELECT ON V_$ARCHIVED_LOG TO C##MONIT;
GRANT SELECT ON GV_$MANAGED_STANDBY TO C##MONIT;
GRANT SELECT ON GV_$STANDBY_LOG TO C##MONIT;
GRANT SELECT ON V_$TEMPFILE TO C##MONIT;
GRANT SELECT ON V_$UNDOSTAT TO C##MONIT;
GRANT SELECT ON V_$RESOURCE_LIMIT TO C##MONIT;
GRANT SELECT ON V_$SYSSTAT TO C##MONIT;
GRANT SELECT ON V_$WAITCLASSMETRIC TO C##MONIT;
GRANT SELECT ON V_$SYSTEM_WAIT_CLASS TO C##MONIT;
GRANT SELECT ON V_$SYSMETRIC TO C##MONIT;
GRANT SELECT ON V_$PROCESS TO C##MONIT;
GRANT SELECT ON GV_$PROCESS TO C##MONIT;
GRANT SELECT ON V_$PGASTAT TO C##MONIT;
GRANT SELECT ON V_$SGASTAT TO C##MONIT;
GRANT SELECT ON V_$SESSION TO C##MONIT;
GRANT SELECT ON V_$SQL TO C##MONIT;
GRANT SELECT ON V_$OSSTAT TO C##MONIT;
GRANT SELECT ON V_$PARAMETER TO C##MONIT;
GRANT SELECT ON GV_$PARAMETER TO C##MONIT;
GRANT SELECT ON V_$SPPARAMETER TO C##MONIT;
GRANT SELECT ON V_$TABLESPACE TO C##MONIT;
GRANT SELECT ON V_$DATAFILE TO C##MONIT;
GRANT SELECT ON V_$ASM_DISKGROUP_STAT TO C##MONIT;
GRANT SELECT ON V_$ASM_DISK_STAT TO C##MONIT;
GRANT SELECT ON V_$ASM_ALIAS TO C##MONIT;
GRANT SELECT ON V_$ASM_FILE TO C##MONIT;
GRANT SELECT ON V_$ASM_DISKGROUP TO C##MONIT;
GRANT SELECT ON V_$ASM_DISK TO C##MONIT;
GRANT SELECT ON DBA_DATA_FILES TO C##MONIT;
GRANT SELECT ON DBA_TABLESPACES TO C##MONIT;
GRANT SELECT ON CDB_DATA_FILES TO C##MONIT;
GRANT SELECT ON CDB_TABLESPACES TO C##MONIT;
GRANT SELECT ON CDB_FREE_SPACE TO C##MONIT;
GRANT SELECT ON CDB_TEMP_FILES TO C##MONIT;
GRANT SELECT ON V_$SORT_SEGMENT TO C##MONIT;
GRANT SELECT ON GV_$SORT_SEGMENT TO C##MONIT;
GRANT SELECT ON DBA_INDEXES TO C##MONIT;
GRANT SELECT ON DBA_OBJECTS TO C##MONIT;
GRANT SELECT ON DBA_TABLES TO C##MONIT;
GRANT SELECT ON DBA_TAB_STATISTICS TO C##MONIT;
GRANT SELECT ON DBA_TAB_COLUMNS TO C##MONIT;
GRANT SELECT ON USER_SEGMENTS TO C##MONIT;
GRANT SELECT ON V_$PDBS TO C##MONIT;
GRANT SELECT ON GV_$PDBS TO C##MONIT;
GRANT SELECT ON V_$CONTAINERS TO C##MONIT;
GRANT SELECT ON GV_$CONTAINERS TO C##MONIT;
GRANT SELECT ON DBA_FREE_SPACE TO C##MONIT;
GRANT SELECT ON DBA_FEATURE_USAGE_STATISTICS TO C##MONIT;
GRANT SELECT ON CDB_FEATURE_USAGE_STATISTICS TO C##MONIT;
EXIT;