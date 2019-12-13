# Test Results
Started: Fri, 13 Dec 2019 09:20:14 -0500

Test                                    |Result|Duration|CPU Avg%|CPU Max%|RAM Avg MiB|RAM Max MiB|Sent Items|Received Items|
----------------------------------------|------|-------:|-------:|-------:|----------:|----------:|---------:|-------------:|
IdleMode                                |PASS  |     15s|     1.3|     4.6|         17|         21|         0|             0|
MetricNoBackend10kDPSOpenCensus         |PASS  |     15s|    19.9|    22.2|         23|         28|    149940|             0|
Metric10kDPS/OpenCensus                 |PASS  |     18s|     9.6|    11.3|         26|         33|    149900|        149900|
Trace10kSPS/JaegerReceiver              |PASS  |     16s|    28.9|    31.5|         46|         56|    148830|        148830|
Trace10kSPS/OpenCensusReceiver          |PASS  |     16s|    27.8|    30.1|         38|         46|    149340|        149340|
TraceNoBackend10kSPSJaeger              |PASS  |     15s|    25.7|    28.1|         99|        138|    148690|             0|
Trace1kSPSWithAttrs/0*0bytes            |PASS  |     15s|    16.8|    19.3|         22|         27|     15000|         15000|
Trace1kSPSWithAttrs/100*50bytes         |PASS  |     15s|    59.9|    65.0|         24|         30|     13920|         13920|
Trace1kSPSWithAttrs/10*1000bytes        |PASS  |     15s|    49.0|    59.4|         24|         30|     14370|         14370|
Trace1kSPSWithAttrs/20*5000bytes        |PASS  |     15s|   108.2|   114.1|         38|         53|     14730|         14730|
TraceBallast1kSPSWithAttrs/0*0bytes     |PASS  |     15s|    16.7|    18.4|         85|        136|     15000|         15000|
TraceBallast1kSPSWithAttrs/100*50bytes  |PASS  |     15s|    41.0|    47.6|        628|        975|     13900|         13900|
TraceBallast1kSPSWithAttrs/10*1000bytes |PASS  |     15s|    36.3|    40.3|        448|        757|     14910|         14910|
TraceBallast1kSPSWithAttrs/20*5000bytes |PASS  |     15s|    77.2|    84.5|        879|       1077|     14070|         14070|
TraceBallast1kSPSAddAttrs/0*0bytes      |PASS  |     15s|    17.1|    18.2|         90|        147|     15000|         15000|
TraceBallast1kSPSAddAttrs/100*50bytes   |PASS  |     15s|    47.1|    49.3|        676|        979|     14820|         14820|
TraceBallast1kSPSAddAttrs/10*1000bytes  |PASS  |     15s|    37.6|    40.0|        516|        838|     15000|         15000|
TraceBallast1kSPSAddAttrs/20*5000bytes  |PASS  |     15s|    53.8|    69.0|        823|       1049|     11740|         11740|

Total duration: 278s
