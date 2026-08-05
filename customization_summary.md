# OpenTelemetry Collector 深度定制化开发技术报告

## 1. 项目背景与挑战
在针对国产数据库 (Sundb) 及特定服务器环境的监控建设中，标准的 OpenTelemetry Collector 组件无法满足以下核心需求：
*   **认证机制复杂**: 目标系统要求基于 MD5 签名的动态 Token 认证，且 Token 具有短时效性（6小时），标准 HTTP Client 仅支持静态 Token。
*   **指标映射僵化**: 标准 `sqlreceiver` 假设“一行数据 = 一个指标”，无法应对复杂 SQL 查询（如 Oracle/Sundb 的缓冲区统计）返回的多维聚合数据。
*   **系统指标定制**: 通用 `hostmetrics` 接收器生成的指标名称、计算公式与现有运维规范不符，且无法精确控制数据存储表结构。
*   **环境兼容性**: 目标服务器（CentOS 旧版本）存在动态链接库版本过低问题，导致标准构建的二进制无法运行。

为此，本项目进行了深度的源码级定制开发，实现了全自动化、高可靠的监控采集探针。

---

## 2. 核心组件开发：SQL Receiver (数据库层)

### 2.1 动态认证与自愈架构 (Dynamic Auth & Self-Healing)
为解决“手动配置 Token 频繁失效”的痛点，我们在 `receiver` 层实现了智能认证状态机：

*   **自动登录 (Auto-Login)**:
    *   启动时，Receiver 主动向配置的 `auth_endpoint` 发起 POST 请求，携带 MD5 加密后的凭证。
    *   代码实现采用了 `net/http` 原生调用，绕过 Collector 框架限制。
*   **启动自愈 (Startup Retry Strategy)**:
    *   **问题**: 启动瞬间若认证服务网络抖动，传统逻辑会直接报错并进入长周期（6h）等待，导致数小时监控盲区。
    *   **优化**: 引入 `Startup Retry Loop`。若未获取首个 Token，进入 **"疯狂重试模式" (每 30s 尝试一次)**，直至成功。
*   **长周期保活 (Token Refresh)**:
    *   内置 `Ticker` 协程，**每 6 小时** 主动刷新 Token，实现 7x24 小时无人值守运行。
    *   Token 存储于内存并通过 `sync.RWMutex` 保护，确保并发查询时的线程安全。
*   **请求头注入**:
    *   在执行 SQL 查询前，中间件层自动读取最新 Token 并封装为 `Authorization: Bearer <token>` Header。

### 2.2 1-to-N 高级指标映射 (Advanced Metric Mapping)
重构了核心解析逻辑 `rowsToMetrics`，使其支持复杂的维度拆解：

*   **Logic Refactor**:
    *   **Before**: `Row -> Single Metric` (一对一)
    *   **After**: `Row -> []MetricConfig -> []Metrics` (一对多)
*   **场景**: 一次查询 `SELECT pool_size, hit_ratio FROM buffer_stat` 即可同时生成 `db.buffer.pool_size` 和 `db.buffer.hit_ratio` 两个独立指标。
*   **优势**: 将数据库查询频率降低 50% 以上，显著减轻对生产数据库的性能压力。

### 2.3 身份元数据自动注入
所有采集出的指标自动附带以下 Tag，无需 SQL 显式返回：
*   `host.name`: 自动获取物理机 Hostname。
*   `service.name`: 统一标准化为 `sundb`。
*   `deployment.instance`: 从 `config.yaml` 动态读取实例名 (如 `sundb-119`)，支持多实例部署区分。

---

## 3. 核心组件开发：Sys Receiver (操作系统层)

为满足特定的运维计算公式（如 "Read KB = 扇区差值 / 2"）和严格的表名规范，从零开发了 `sysreceiver` 组件。采用 **微内核 + 插件式** 架构。

### 3.1 插件化采集器 (Scrapers)
实现了 5 个独立的原子采集模块，互不干扰：

| 模块名称 | 数据源 | 核心逻辑与公式 | 亮点 |
| :--- | :--- | :--- | :--- |
| **CPU** | `/proc/stat` | `Usage = (Busy2 - Busy1) / (Total2 - Total1)` | 采样差值计算，精确反映区间负载 |
| **Memory** | `/proc/meminfo` | `Used = Total - Free - Buffers - Cached` | 剔除 Buffer/Cache，反映真实应用内存占用 |
| **Disk** | `syscall.Statfs` | 直接调用系统底层的 Statfs 接口 | 无需执行 `df` 命令，极低开销 |
| **Disk IO** | `/proc/diskstats` | `ReadKB = (SectorsRead_Diff) * 512 / 1024` | **1秒两次采样法**，精确计算瞬时速率 |
| **Net IO** | `/proc/net/dev` | `InRate = (RxBytes_Diff) / 1024` | 自动过滤 `lo` 等回环网卡，只关注物理流量 |

### 3.2 统一数据模型 (Schema Unification)
解决了早期方案中“按节点和日期分表”导致表数量失控的问题。

*   **统一表名**: 强制规范为 `sys_cpu_usage`, `sys_memory_usage`, `sys_disk_io_rate` 等静态表名。
*   **维度隔离**: 引入 `NODE` 标签列 (如 `G1N1`)。所有节点数据写入同一张表，通过 TAG 区分。
*   **GreptimeDB 适配**: 针对时序数据库特性，自动处理时间戳对齐 (Timestamp Alignment) 和大小写敏感性问题。

### 3.3 健壮性设计 (Defensive Programming)
*   **脏数据清洗**: 在解析 `/proc` 文件时增加了严格的 `strconv` 错误检查。若内核文件输出乱码或格式异常（如 Disk IO 统计行被截断），自动跳过本次采集，**防止 0 值或脏数据污染数据库**。
*   **热拔插保护**: 动态检测磁盘/网卡设备。若设备在运行中被物理移除，采集器会自动感知并停止对该设备的采集，不会引发进程 Panic。

---

## 4. 工程化与运维交付

### 4.1 静态编译 (Static Linking)
*   **挑战**: 在 CentOS 7 环境运行标准构建的 Collector 时，报 `203/EXEC` 错误，原因是 `glibc` 版本过低或动态加载器路径不匹配。
*   **解决**: 使用 `CGO_ENABLED=0` 环境变量进行全静态编译。
*   **成果**: 生成的 `otelcorecol_linux` 二进制文件不依赖任何系统动态库，实现了 **"Copy & Run"**，兼容所有 Linux 发行版。

### 4.2 Systemd 服务化
*   交付了标准的 `otelcol.service` 配置文件。
*   支持 `systemctl start/stop/status` 管理。
*   配置了 `Restart=always` 策略，确保进程在意外退出后自动拉起，保障监控连续性。
