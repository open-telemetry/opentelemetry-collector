# Test Results
Started: Wed, 03 Jul 2019 05:42:19 +0530

Test                                    |Result|Duration|CPU Avg%|CPU Max%|RAM Avg MiB|RAM Max MiB|Sent Spans|Received Spans
----------------------------------------|------|-------:|-------:|-------:|----------:|----------:|---------:|-------------:
TestIdleMode                            |PASS  |     10s|     0.3|     1.0|         25|         34|         0|             0
Test10kSPS                              |PASS  |     16s|    72.6|    92.0|         34|         41|    144520|        144520
TestNoBackend10kSPS                     |PASS  |     10s|    75.6|    78.3|         29|         40|     97460|             0
Test1000SPSWithAttributes/0*0bytes      |PASS  |     11s|    22.3|    24.0|         31|         42|     10000|         10000
Test1000SPSWithAttributes/100*50bytes   |PASS  |     11s|    92.3|   106.3|         33|         46|      9990|          9990
Test1000SPSWithAttributes/10*1000bytes  |PASS  |     11s|    86.2|    93.3|         34|         46|     10000|         10000
Test1000SPSWithAttributes/20*5000bytes  |PASS  |     11s|   126.3|   130.0|         47|         64|      9990|          9990
TestBallast1000SPSWithAttributes/0*0bytes|PASS  |     11s|    20.4|    21.7|         71|        109|     10000|         10000
TestBallast1000SPSWithAttributes/100*50bytes|PASS  |     11s|    64.4|    68.0|        492|        845|      9950|          9950
TestBallast1000SPSWithAttributes/10*1000bytes|PASS  |     11s|    55.1|    59.7|        350|        575|      9990|          9990
TestBallast1000SPSWithAttributes/20*5000bytes|PASS  |     11s|    67.8|    69.3|        786|       1081|      9950|          9950

Total duration: 127s
