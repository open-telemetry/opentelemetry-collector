# Sundb 数据库可观测性监控指标手册

本文档详细描述了基于 OpenTelemetry Collector 定制开发的 Sundb 数据库监控体系，涵盖所有可监控指标的含义、数据来源、查询 SQL、采集频率、OTel 改造设计以及 API 接口与输出格式。

---

## 一、总体架构概览

```
┌─────────────┐     HTTP/JSON      ┌──────────────────┐    OTLP/HTTP     ┌────────────┐
│  Sundb 数据库 │ ◄──────────────── │  OTel Collector   │ ──────────────► │ GreptimeDB  │
│  (9989端口)   │    SQL查询+JWT    │  (定制化二进制)     │                  │ (时序存储)   │
└─────────────┘                    │                    │    OTLP/Proto    ┌────────────┐
                                   │  ┌─sqlreceiver    │ ──────────────► │   Kafka     │
┌─────────────┐                    │  ├─sysreceiver    │                  │ (消息队列)   │
│ /proc 文件系统│ ◄──── 直接读取 ──── │  └─filelogreceiver│                  └────────────┘
└─────────────┘                    └──────────────────┘
```

### 核心组件说明

| 组件              | 类型                | 职责                                              |
| :---------------- | :------------------ | :------------------------------------------------ |
| `sqlreceiver`     | Receiver (定制开发) | 通过 HTTP API 执行 SQL 查询采集数据库内部指标     |
| `sysreceiver`     | Receiver (定制开发) | 直接读取 `/proc` 文件系统采集操作系统指标         |
| `filelogreceiver` | Receiver (社区组件) | 采集 Sundb 的 `system.trc` 和 `listener.trc` 日志 |

### 采集频率

| 采集器            | 默认频率              | 配置项                     |
| :---------------- | :-------------------- | :------------------------- |
| `sqlreceiver`     | **30 秒**             | `collection_interval: 30s` |
| `sysreceiver`     | **30 秒**             | `collection_interval: 30s` |
| `filelogreceiver` | **实时** (200ms 轮询) | `poll_interval: 200ms`     |

---

## 二、数据库层指标 (sqlreceiver)

### 2.1 API 接口与认证

#### 查询接口

| 项目             | 值                                  |
| :--------------- | :---------------------------------- |
| **Endpoint**     | `http://localhost:9989/query`       |
| **Method**       | `POST`                              |
| **Content-Type** | `application/json`                  |
| **请求体格式**   | `{"sql": "<SQL语句>"}`              |
| **认证方式**     | `Authorization: Bearer <JWT Token>` |

#### 认证接口 (动态 Token)

| 项目             | 值                                                           |
| :--------------- | :----------------------------------------------------------- |
| **Endpoint**     | `http://127.0.0.1:9989/auth/login`                           |
| **Method**       | `POST`                                                       |
| **Content-Type** | `application/x-www-form-urlencoded`                          |
| **请求体**       | `password=21232f297a57a5a743894a0e4a801fc3` (MD5 of "admin") |
| **刷新周期**     | 每 **6 小时** 自动刷新                                       |
| **启动重试**     | 失败时每 **30 秒** 重试，直至成功                            |

#### 认证响应示例

```json
{
    "retcode": 0,
    "retmesg": "Authentication pass",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### SQL 查询响应格式

```json
{
    "retcode": 0,
    "retmesg": "ok",
    "result": "{\"COL_NAMES\":[{\"COLUMN1\":\"VARCHAR\"}, ...], \"VALUES\":[{\"COLUMN1\":\"value1\", ...}, ...]}"
}
```

> 注意：`result` 字段是一个 **JSON 字符串**（需要二次反序列化），内部包含 `COL_NAMES`（列名与类型）和 `VALUES`（行数据数组）。

---

### 2.2 数据库指标详解

#### 2.2.1 缓冲区缓存 (BUFFERCACHE)

| 项           | 说明                  |
| :----------- | :-------------------- |
| **监控编号** | monitors.BUFFERCACHE  |
| **数据源表** | `X$BUFFER_STAT@local` |
| **查询频率** | 30s                   |

**SQL 语句：**

```sql
SELECT
    nvl(buff.CLUSTER_MEMBER_NAME, 'STANDALONE') AS "MEMBER_NAME",
    CAST(buff.BUFFER_POOL_SIZE AS NUMBER) * 8 / 1024 AS "POOL_SIZE",
    CAST(buff.HOT_REGION_PERCENTAGE AS NUMBER) AS "LRU_HOT_REGION",
    round(CAST((BUFFER_HIT / (BUFFER_HIT + BUFFER_MISS) * 100) AS number), 2) AS "BUFFER_HIT",
    CAST(buff.TOTAL_WRITES AS NUMBER) AS "TOTAL_WRITES"
FROM X$BUFFER_STAT@local AS buff
```

**生成的指标：**

| OTel 指标名              | GreptimeDB 表名          | 值来源字段       | 类型  | 含义               | 维度标签 |
| :----------------------- | :----------------------- | :--------------- | :---- | :----------------- | :------- |
| `db.buffer.pool_size`    | `db_buffer_pool_size`    | `POOL_SIZE`      | Gauge | 缓冲池大小 (MB)    | `member` |
| `db.buffer.hot_region`   | `db_buffer_hot_region`   | `LRU_HOT_REGION` | Gauge | LRU 热区百分比 (%) | `member` |
| `db.buffer.hit_ratio`    | `db_buffer_hit_ratio`    | `BUFFER_HIT`     | Gauge | 缓冲命中率 (%)     | `member` |
| `db.buffer.total_writes` | `db_buffer_total_writes` | `TOTAL_WRITES`   | Sum   | 总写入次数 (累计)  | `member` |

---

#### 2.2.6 索引使用 (INDEX)

| 项           | 说明           |
| :----------- | :------------- |
| **监控编号** | monitors.INDEX |
| **数据源表** | `TECH_INDEX`   |
| **查询频率** | 30s            |

**SQL 语句：**

```sql
SELECT OWNER, TAB_SCHEMA, TAB_NAME, IDX_NAME, TBS_NAME, USE_MBYTE, PERCENTAGE
FROM TECH_INDEX
ORDER BY PERCENTAGE DESC
LIMIT 5
```

**生成的指标：**

| OTel 指标名           | 值来源字段   | 类型  | 含义               | 维度标签                                          |
| :-------------------- | :----------- | :---- | :----------------- | :------------------------------------------------ |
| `db.index.use_mbyte`  | `USE_MBYTE`  | Gauge | 索引占用空间 (MB)  | `owner`, `schema`, `table`, `index`, `tablespace` |
| `db.index.percentage` | `PERCENTAGE` | Gauge | 索引空间使用率 (%) | `owner`, `schema`, `table`, `index`, `tablespace` |

---

#### 2.2.11 私有静态区 (PSA)

| 项           | 说明                                                         |
| :----------- | :----------------------------------------------------------- |
| **监控编号** | monitors.PSA                                                 |
| **数据源表** | `X$KN_PROC_STAT@local`, `X$KN_PROC_ENV@local`, `X$PROPERTY@local`, `X$SESSION@local` |
| **查询频率** | 30s                                                          |

**SQL 语句：**

```sql
SELECT
    NVL(XS.CLUSTER_MEMBER_NAME, 'STANDALONE') AS "MEMBER_NAME",
    XS.ID AS "ID", XS.SERIAL AS "SERIAL", XS.PROGRAM AS "PROGRAM",
    XP.VALUE / 1024 / 1024 AS "TOTAL_PSA_MEGA",
    SUM(XKPS.VALUE) / 1024 / 1024 AS "ALLOC_PSA_MEGA",
    (SUM(XKPS.VALUE) / XP.VALUE) * 100 AS "ALLOC_PERCENT"
FROM X$KN_PROC_STAT@local XKPS, X$KN_PROC_ENV@local XKPE,
     X$PROPERTY@local XP, X$SESSION@local XS
WHERE XKPS.ID = XKPE.ID AND XKPE.OS_PROC_ID = XS.SERVER_PROCESS
  AND XP.PROPERTY_NAME = 'PRIVATE_STATIC_AREA_SIZE'
  AND XS.TOP_LAYER != 12 AND XS.PROGRAM != 'cluster peer'
  AND XKPS.NAME LIKE '%TOTAL%'
GROUP BY XS.CLUSTER_MEMBER_NAME, XP.VALUE, XS.ID, XS.SERIAL, XS.PROGRAM
ORDER BY "MEMBER_NAME", XS.ID
```

**生成的指标：**

| OTel 指标名            | 值来源字段       | 类型  | 含义                | 维度标签                   |
| :--------------------- | :--------------- | :---- | :------------------ | :------------------------- |
| `db.psa.total_mega`    | `TOTAL_PSA_MEGA` | Gauge | PSA 总大小 (MB)     | `member`, `program`, `sid` |
| `db.psa.alloc_mega`    | `ALLOC_PSA_MEGA` | Gauge | PSA 已分配大小 (MB) | `member`, `program`, `sid` |
| `db.psa.alloc_percent` | `ALLOC_PERCENT`  | Gauge | PSA 分配率 (%)      | `member`, `program`, `sid` |

---

#### 2.2.12 查询频率统计 (QPS)

| 项           | 说明                      |
| :----------- | :------------------------ |
| **监控编号** | monitors.QPS              |
| **数据源表** | `V$SYSTEM_SQL_STAT@local` |
| **查询频率** | 30s                       |

**SQL 语句：**

```sql
SELECT t.*, TO_CHAR(SYSTIMESTAMP, 'YYYY-MM-DD HH24:MI:SS') AS mon_time
FROM V$SYSTEM_SQL_STAT@local t
WHERE STAT_NAME = 'COMMAND: SELECT'
```

**生成的指标：**

| OTel 指标名     | 值来源字段   | 类型 | 含义                | 维度标签                |
| :-------------- | :----------- | :--- | :------------------ | :---------------------- |
| `db.qps.select` | `STAT_VALUE` | Sum  | SELECT 累计执行次数 | `stat_name`, `comments` |

---

#### 2.2.13 Redo Log 状态 (REDOLOG_SWITCH)

| 项           | 说明                             |
| :----------- | :------------------------------- |
| **监控编号** | monitors.REDOLOG_SWITCH          |
| **数据源表** | `FIXED_TABLE_SCHEMA.X$LOG_GROUP` |
| **查询频率** | 30s                              |

**SQL 语句：**

```sql
SELECT * FROM FIXED_TABLE_SCHEMA.X$LOG_GROUP
```

**生成的指标：**

| OTel 指标名               | 值来源字段     | 类型  | 含义                      | 维度标签                         |
| :------------------------ | :------------- | :---- | :------------------------ | :------------------------------- |
| `db.redolog.file_size`    | `FILE_SIZE`    | Gauge | Redo Log 文件大小 (Bytes) | `group_id`, `group_ord`, `state` |
| `db.redolog.member_count` | `MEMBER_COUNT` | Gauge | Redo Log 组成员数         | `group_id`, `group_ord`, `state` |

---

#### 2.2.14 会话统计 (SESSION)

| 项           | 说明             |
| :----------- | :--------------- |
| **监控编号** | monitors.SESSION |
| **数据源表** | `V$SESSION`      |
| **查询频率** | 30s              |

**SQL 语句：**

```sql
SELECT count(*) AS COUNT FROM V$SESSION
```

**生成的指标：**

| OTel 指标名        | 值来源字段 | 类型  | 含义           | 维度标签 |
| :----------------- | :--------- | :---- | :------------- | :------- |
| `db.session.count` | `COUNT`    | Gauge | 当前活跃会话数 | 无       |

---

#### 2.2.15 / 2.2.19 SQL 执行统计与 TPS (SQL & TPS)

| 项           | 说明                        |
| :----------- | :-------------------------- |
| **监控编号** | monitors.SQL & monitors.TPS |
| **数据源表** | `gv$system_sql_stat`        |
| **查询频率** | 30s                         |

**SQL 语句：**

```sql
SELECT t.*, to_char(SYSTIMESTAMP, 'YYYY-MM-DD HH24:MI:SS') AS mon_time
FROM gv$system_sql_stat t
WHERE STAT_NAME LIKE 'COMMAND: INSERT%'
   OR STAT_NAME LIKE 'COMMAND: UPDATE%'
   OR STAT_NAME LIKE 'COMMAND: DELETE%'
   OR STAT_NAME = 'COMMAND: SELECT'
```

**生成的指标：**

| OTel 指标名    | 值来源字段   | 类型 | 含义                  | 维度标签                     |
| :------------- | :----------- | :--- | :-------------------- | :--------------------------- |
| `db.sql.stats` | `STAT_VALUE` | Sum  | 各类 SQL 累计执行次数 | `origin_member`, `stat_name` |

> 通过 `stat_name` 维度可区分 INSERT/UPDATE/DELETE/SELECT。

---

#### 2.2.17 共享静态区 (SSA)

| 项           | 说明                     |
| :----------- | :----------------------- |
| **监控编号** | monitors.SSA             |
| **数据源表** | `X$KN_SYSTEM_INFO@LOCAL` |
| **查询频率** | 30s                      |

**SQL 语句：**

```sql
SELECT
    NVL(CLUSTER_GROUP_NAME, 'dummy') GROUP_NAME,
    NVL(CLUSTER_MEMBER_NAME, 'dummy') MEMBER_NAME,
    NAME, VALUE
FROM X$KN_SYSTEM_INFO@LOCAL
WHERE NAME IN ('VARIABLE_STATIC_TOTAL_SIZE', 'VARIABLE_STATIC_ALLOC_SIZE')
```

**生成的指标：**

| OTel 指标名    | 值来源字段 | 类型  | 含义                      | 维度标签                            |
| :------------- | :--------- | :---- | :------------------------ | :---------------------------------- |
| `db.ssa.value` | `VALUE`    | Gauge | SSA 总量/已分配量 (Bytes) | `group_name`, `member_name`, `name` |

> 通过 `name` 维度区分 `VARIABLE_STATIC_TOTAL_SIZE` 和 `VARIABLE_STATIC_ALLOC_SIZE`。

---

#### 2.2.18 表空间使用 (TABLESPACE)

| 项           | 说明                    |
| :----------- | :---------------------- |
| **监控编号** | monitors.TABLESPACE     |
| **数据源表** | `TECH_TABLESPACE@LOCAL` |
| **查询频率** | 30s                     |

**SQL 语句：**

```sql
SELECT
    CLUSTER_NAME AS "MEMBER_NAME",
    TABLESPACE_NAME, TOTAL_MEGABYTE, USED_MEGABYTE,
    FREE_MEGABYTE, USED_PERCENTAGE
FROM TECH_TABLESPACE@LOCAL
```

**生成的指标：**

| OTel 指标名                  | 值来源字段        | 类型  | 含义             | 维度标签               |
| :--------------------------- | :---------------- | :---- | :--------------- | :--------------------- |
| `db.tablespace.total_mb`     | `TOTAL_MEGABYTE`  | Gauge | 表空间总量 (MB)  | `member`, `tablespace` |
| `db.tablespace.used_mb`      | `USED_MEGABYTE`   | Gauge | 表空间已用 (MB)  | `member`, `tablespace` |
| `db.tablespace.free_mb`      | `FREE_MEGABYTE`   | Gauge | 表空间剩余 (MB)  | `member`, `tablespace` |
| `db.tablespace.used_percent` | `USED_PERCENTAGE` | Gauge | 表空间使用率 (%) | `member`, `tablespace` |

---

#### 2.2.20 事务统计 (TRANSACTION)

| 项           | 说明                 |
| :----------- | :------------------- |
| **监控编号** | monitors.TRANSACTION |
| **数据源表** | `v$transaction`      |
| **查询频率** | 30s                  |

**SQL 语句：**

```sql
SELECT count(distinct TRANS_ID) "TRANS_COUNT" FROM v$transaction
```

**生成的指标：**

| OTel 指标名            | 值来源字段    | 类型  | 含义           | 维度标签 |
| :--------------------- | :------------ | :---- | :------------- | :------- |
| `db.transaction.count` | `TRANS_COUNT` | Gauge | 当前活跃事务数 | 无       |

---

### 2.3 数据库指标汇总表

| 指标编号    | OTel 指标名                  | 数据源表             | 类型  | 含义             |
| :---------- | :--------------------------- | :------------------- | :---- | :--------------- |
| BUFFERCACHE | `db.buffer.pool_size`        | `X$BUFFER_STAT`      | Gauge | 缓冲池大小 (MB)  |
| BUFFERCACHE | `db.buffer.hot_region`       | `X$BUFFER_STAT`      | Gauge | LRU 热区 (%)     |
| BUFFERCACHE | `db.buffer.hit_ratio`        | `X$BUFFER_STAT`      | Gauge | 命中率 (%)       |
| BUFFERCACHE | `db.buffer.total_writes`     | `X$BUFFER_STAT`      | Sum   | 总写入次数       |
| INDEX       | `db.index.use_mbyte`         | `TECH_INDEX`         | Gauge | 索引空间 (MB)    |
| INDEX       | `db.index.percentage`        | `TECH_INDEX`         | Gauge | 索引使用率 (%)   |
| PSA         | `db.psa.total_mega`          | `X$KN_PROC_STAT` 等  | Gauge | PSA 总量 (MB)    |
| PSA         | `db.psa.alloc_mega`          | `X$KN_PROC_STAT` 等  | Gauge | PSA 已分配 (MB)  |
| PSA         | `db.psa.alloc_percent`       | `X$KN_PROC_STAT` 等  | Gauge | PSA 分配率 (%)   |
| QPS         | `db.qps.select`              | `V$SYSTEM_SQL_STAT`  | Sum   | SELECT 执行次数  |
| REDOLOG     | `db.redolog.file_size`       | `X$LOG_GROUP`        | Gauge | Redo 文件大小    |
| REDOLOG     | `db.redolog.member_count`    | `X$LOG_GROUP`        | Gauge | Redo 组成员数    |
| SESSION     | `db.session.count`           | `V$SESSION`          | Gauge | 活跃会话数       |
| SQL/TPS     | `db.sql.stats`               | `gv$system_sql_stat` | Sum   | SQL 执行统计     |
| SSA         | `db.ssa.value`               | `X$KN_SYSTEM_INFO`   | Gauge | 共享静态区大小   |
| TABLESPACE  | `db.tablespace.total_mb`     | `TECH_TABLESPACE`    | Gauge | 表空间总量 (MB)  |
| TABLESPACE  | `db.tablespace.used_mb`      | `TECH_TABLESPACE`    | Gauge | 表空间已用 (MB)  |
| TABLESPACE  | `db.tablespace.free_mb`      | `TECH_TABLESPACE`    | Gauge | 表空间剩余 (MB)  |
| TABLESPACE  | `db.tablespace.used_percent` | `TECH_TABLESPACE`    | Gauge | 表空间使用率 (%) |
| TRANSACTION | `db.transaction.count`       | `v$transaction`      | Gauge | 活跃事务数       |

---

## 三、操作系统层指标 (sysreceiver)

### 3.1 采集方式说明

`sysreceiver` **不通过 HTTP API**，而是直接读取 Linux `/proc` 伪文件系统和 `syscall` 系统调用。无需认证，无网络开销。

- **采集频率**: 30 秒
- **采样方法**: CPU、DiskIO、NetIO 采用"**1 秒两次采样求差值**"的方式计算瞬时速率

### 3.2 操作系统指标详解

#### 3.2.1 CPU 使用率

| 项                  | 说明                      |
| :------------------ | :------------------------ |
| **GreptimeDB 表名** | `sys_cpu_usage`           |
| **数据源**          | `/proc/stat`              |
| **采样方式**        | 间隔 1 秒读取两次，取差值 |

**计算公式：**

```
Total = User + Nice + System + Idle + IOWait + IRQ + SoftIRQ + Steal
Busy  = Total - Idle - IOWait
Usage = (Busy2 - Busy1) / (Total2 - Total1) × 100%
```

**指标属性：**

| 列名                 | 来源              | 含义                 |
| :------------------- | :---------------- | :------------------- |
| `greptime_value`     | 计算值            | CPU 使用率 (%)       |
| `NODE`               | 配置 `node_value` | 节点标识 (如 "G1N1") |
| `HOST_IP`            | 配置 `host_ip`    | 主机 IP              |
| `greptime_timestamp` | 自动生成          | 采集时间戳           |

---

#### 3.2.2 内存使用率

| 项                  | 说明               |
| :------------------ | :----------------- |
| **GreptimeDB 表名** | `sys_memory_usage` |
| **数据源**          | `/proc/meminfo`    |
| **采样方式**        | 直接读取           |

**计算公式：**

```
Used = MemTotal - MemFree - Buffers - Cached
Usage = Used / MemTotal × 100%
```

**指标属性：**

| 列名             | 来源   | 含义           |
| :--------------- | :----- | :------------- |
| `greptime_value` | 计算值 | 内存使用率 (%) |
| `NODE`           | 配置   | 节点标识       |
| `HOST_IP`        | 配置   | 主机 IP        |

---

#### 3.2.3 磁盘使用率

| 项                  | 说明                  |
| :------------------ | :-------------------- |
| **GreptimeDB 表名** | `sys_disk_usage`      |
| **数据源**          | `syscall.Statfs("/")` |
| **采样方式**        | 直接调用系统接口      |

**计算公式：**

```
Total = Blocks × BlockSize
Free  = BlocksFree × BlockSize
Used  = Total - Free
Usage = Used / Total × 100%
```

**指标属性：**

| 列名             | 来源   | 含义           |
| :--------------- | :----- | :------------- |
| `greptime_value` | 计算值 | 磁盘使用率 (%) |
| `NODE`           | 配置   | 节点标识       |
| `HOST_IP`        | 配置   | 主机 IP        |

---

#### 3.2.4 磁盘 IO (三个表)

| 项                  | 说明                                                         |
| :------------------ | :----------------------------------------------------------- |
| **GreptimeDB 表名** | `sys_disk_io_rate` / `sys_disk_io_read_kb` / `sys_disk_io_write_kb` |
| **数据源**          | `/proc/diskstats`                                            |
| **采样方式**        | 间隔 1 秒读取两次，取差值                                    |
| **设备过滤**        | 仅采集 `sd*`, `vd*`, `nvme*` 开头的设备                      |

**计算公式：**

```
IO Rate  = (IoTime2 - IoTime1) / 1000  (秒)
Read KB  = (SectorsRead2 - SectorsRead1) × 512 / 1024
Write KB = (SectorsWritten2 - SectorsWritten1) × 512 / 1024
```

**指标属性：**

| 列名             | 来源   | 含义                             |
| :--------------- | :----- | :------------------------------- |
| `greptime_value` | 计算值 | IO 速率/读写量                   |
| `NODE`           | 配置   | 节点标识                         |
| `HOST_IP`        | 配置   | 主机 IP                          |
| `DEVICE`         | 设备名 | 磁盘设备名 (如 `sda`, `nvme0n1`) |

---

#### 3.2.5 网络 IO (两个表)

| 项                  | 说明                               |
| :------------------ | :--------------------------------- |
| **GreptimeDB 表名** | `sys_net_io_in` / `sys_net_io_out` |
| **数据源**          | `/proc/net/dev`                    |
| **采样方式**        | 间隔 1 秒读取两次，取差值          |
| **过滤规则**        | 排除 `lo` 回环网卡                 |

**计算公式：**

```
InRate  = (RxBytes2 - RxBytes1) / 1024  (KB/s)
OutRate = (TxBytes2 - TxBytes1) / 1024  (KB/s)
```

**指标属性：**

| 列名             | 来源   | 含义            |
| :--------------- | :----- | :-------------- |
| `greptime_value` | 计算值 | 网络流量 (KB/s) |
| `NODE`           | 配置   | 节点标识        |
| `HOST_IP`        | 配置   | 主机 IP         |

---

### 3.3 操作系统指标汇总表

| GreptimeDB 表名        | 数据源            | 采样方式 | 含义              | 关键维度                    |
| :--------------------- | :---------------- | :------- | :---------------- | :-------------------------- |
| `sys_cpu_usage`        | `/proc/stat`      | 差值法   | CPU 使用率 (%)    | `NODE`, `HOST_IP`           |
| `sys_memory_usage`     | `/proc/meminfo`   | 直接读取 | 内存使用率 (%)    | `NODE`, `HOST_IP`           |
| `sys_disk_usage`       | `syscall.Statfs`  | 直接读取 | 磁盘使用率 (%)    | `NODE`, `HOST_IP`           |
| `sys_disk_io_rate`     | `/proc/diskstats` | 差值法   | 磁盘 IO 利用率    | `NODE`, `HOST_IP`, `DEVICE` |
| `sys_disk_io_read_kb`  | `/proc/diskstats` | 差值法   | 磁盘读吞吐 (KB)   | `NODE`, `HOST_IP`, `DEVICE` |
| `sys_disk_io_write_kb` | `/proc/diskstats` | 差值法   | 磁盘写吞吐 (KB)   | `NODE`, `HOST_IP`, `DEVICE` |
| `sys_net_io_in`        | `/proc/net/dev`   | 差值法   | 网络入流量 (KB/s) | `NODE`, `HOST_IP`           |
| `sys_net_io_out`       | `/proc/net/dev`   | 差值法   | 网络出流量 (KB/s) | `NODE`, `HOST_IP`           |

---

## 四、日志采集 (filelogreceiver)

### 4.1 System.trc (数据库系统日志)

| 项           | 说明                                                     |
| :----------- | :------------------------------------------------------- |
| **文件路径** | `/home/sundb/sundb-server-.../sundb_data/trc/system.trc` |
| **存储位置** | GreptimeDB `sundb_logs.sundb_logs` 表                    |
| **采集方式** | 实时尾随读取 (200ms 轮询)                                |

**日志格式示例：**

```
[2025-12-09 07:19:49.379721 INSTANCE(SUNDB) THREAD(15562,126947832010560)] [INFORMATION]
[MODULE_NAME] 日志具体内容...
```

**解析后的字段：**

| 字段        | 来源               | 含义                             |
| :---------- | :----------------- | :------------------------------- |
| `body`      | 正则提取 `message` | 日志正文                         |
| `instance`  | 正则提取           | 数据库实例名                     |
| `thread_id` | 正则提取           | 线程 ID                          |
| `component` | 正则提取           | 模块名                           |
| `severity`  | 正则提取并映射     | 日志级别 (INFO/WARN/ERROR/DEBUG) |

### 4.2 Listener.trc (监听器日志)

| 项           | 说明                                                       |
| :----------- | :--------------------------------------------------------- |
| **文件路径** | `/home/sundb/sundb-server-.../sundb_data/trc/listener.trc` |
| **存储位置** | 同 `sundb_logs.sundb_logs` 表                              |

---

## 五、OTel 改造设计详解

### 5.1 sqlreceiver 核心改造

```
┌───────────────────────────────────────────────────────────┐
│                      sqlreceiver                          │
│                                                           │
│  ┌─────────┐    ┌──────────────┐    ┌────────────────┐   │
│  │ authLoop │───►│ refreshToken │───►│  token (内存)   │   │
│  │ (6h周期) │    │ POST /login  │    │ sync.RWMutex   │   │
│  └─────────┘    └──────────────┘    └───────┬────────┘   │
│                                             │ 读取        │
│  ┌──────────────┐   ┌───────────────┐  ┌────▼─────────┐  │
│  │ queryLoop(Q) │──►│ executeOnce() │─►│ HTTP POST    │  │
│  │ (30s Ticker) │   │ + 3次重试     │  │ Bearer Token │  │
│  └──────────────┘   └───────┬───────┘  └──────────────┘  │
│                             │                             │
│                    ┌────────▼────────┐                    │
│                    │ parseResponse() │                    │
│                    │ 二次JSON反序列化 │                    │
│                    └────────┬────────┘                    │
│                             │                             │
│                    ┌────────▼────────┐                    │
│                    │ rowsToMetrics() │                    │
│                    │ 1-to-N 映射     │                    │
│                    └─────────────────┘                    │
└───────────────────────────────────────────────────────────┘
```

**关键设计点：**

1. **1-to-N 映射**: 单条 SQL 返回的一行数据可同时映射为多个 OTel Metric。
2. **身份注入**: 每个数据点自动附加 `host.name`, `service.name`, `deployment.instance`。
3. **指数退避重试**: 查询失败时最多重试 3 次，延迟 1s → 2s → 4s。

### 5.2 sysreceiver 核心改造

```
┌──────────────────────────────────────────────┐
│                 sysreceiver                   │
│                                               │
│  ┌───────────┐                                │
│  │ scrapeLoop│ (30s Ticker)                   │
│  └─────┬─────┘                                │
│        │ 依次调用                              │
│        ├─► cpuScraper    (/proc/stat)         │
│        ├─► memScraper    (/proc/meminfo)      │
│        ├─► diskScraper   (syscall.Statfs)     │
│        ├─► diskIoScraper (/proc/diskstats)    │
│        └─► netScraper    (/proc/net/dev)      │
│                                               │
│  每个 Scraper 独立容错，失败不影响其他          │
└──────────────────────────────────────────────┘
```

### 5.3 输出格式

所有指标最终以 **OTLP (OpenTelemetry Protocol)** 格式输出，支持两种编码：

| 导出目标             | 协议           | 编码格式     | 端点                                         |
| :------------------- | :------------- | :----------- | :------------------------------------------- |
| GreptimeDB (Metrics) | OTLP/HTTP      | Protobuf     | `http://172.19.19.127:4000/v1/otlp`          |
| GreptimeDB (Logs)    | OTLP/HTTP      | Protobuf     | `http://172.19.19.127:4000/v1/otlp`          |
| Kafka (Metrics)      | Kafka Producer | `otlp_proto` | `172.19.19.117:9092` / topic: `otel-metrics` |
| Kafka (Logs)         | Kafka Producer | `otlp_proto` | `172.19.19.117:9092` / topic: `otel-logs`    |

### 5.4 每个指标数据点的标准结构

```json
{
  "resource": {
    "attributes": {
      "host.name": "g2n2",
      "service.name": "sundb",
      "deployment.instance": "sundb-119"
    }
  },
  "scopeMetrics": [{
    "metrics": [{
      "name": "db.buffer.hit_ratio",
      "gauge": {
        "dataPoints": [{
          "asDouble": 99.85,
          "timeUnixNano": "1735689600000000000",
          "attributes": {
            "member": "STANDALONE"
          }
        }]
      }
    }]
  }]
}
```
