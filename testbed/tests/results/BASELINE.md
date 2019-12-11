# Test Results
Started: Tue, 10 Dec 2019 12:36:08 -0500

Test                                    |Result|Duration|CPU Avg%|CPU Max%|RAM Avg MiB|RAM Max MiB|Sent Items|Received Items|
----------------------------------------|------|-------:|-------:|-------:|----------:|----------:|---------:|-------------:|
IdleMode                                |PASS  |     16s|     1.0|     3.3|         16|         20|         0|             0|
MetricNoBackend10kDPSOpenCensus         |PASS  |     15s|    18.7|    19.7|         21|         26|    149960|             0|
Metric10kDPS/OpenCensus                 |PASS  |     18s|    11.5|    14.3|         27|         33|    149900|        149900|
Trace10kSPS/JaegerReceiver              |PASS  |     15s|    48.9|    52.4|         20|         25|    149990|        149990|
Trace10kSPS/OpenCensusReceiver          |PASS  |     15s|    42.8|    47.4|         21|         26|    149950|        149950|
TraceNoBackend10kSPSJaeger              |PASS  |     15s|    24.4|    27.1|         95|        131|    149310|             0|
Trace1kSPSWithAttrs/0*0bytes            |PASS  |     15s|    18.2|    18.5|         22|         27|     15000|         15000|
Trace1kSPSWithAttrs/100*50bytes         |PASS  |     15s|    64.9|    73.0|         24|         30|     15000|         15000|
Trace1kSPSWithAttrs/10*1000bytes        |PASS  |     15s|    56.3|    61.0|         24|         29|     15000|         15000|
Trace1kSPSWithAttrs/20*5000bytes        |PASS  |     20s|    84.4|   112.1|         66|         77|     14990|         14990|
TraceBallast1kSPSWithAttrs/0*0bytes     |PASS  |     15s|    17.2|    17.9|         84|        134|     15000|         15000|
TraceBallast1kSPSWithAttrs/100*50bytes  |PASS  |     15s|    40.7|    44.0|        655|        976|     15000|         15000|
TraceBallast1kSPSWithAttrs/10*1000bytes |PASS  |     15s|    34.6|    38.1|        449|        762|     15000|         15000|
TraceBallast1kSPSWithAttrs/20*5000bytes |PASS  |     45s|    27.4|    36.1|        969|       1067|     15000|         15000|
TraceBallast1kSPSAddAttrs/0*0bytes      |PASS  |     15s|    17.5|    19.0|         89|        145|     15000|         15000|
TraceBallast1kSPSAddAttrs/100*50bytes   |PASS  |     15s|    43.8|    51.4|        676|        979|     15000|         15000|
TraceBallast1kSPSAddAttrs/10*1000bytes  |PASS  |     15s|    39.2|    43.9|        515|        837|     15000|         15000|
TraceBallast1kSPSAddAttrs/20*5000bytes  |PASS  |     15s|    69.0|    76.9|        848|       1071|     15000|         15000|
