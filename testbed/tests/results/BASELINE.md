# Test Results
Started: Tue, 25 Jun 2019 10:31:56 -0400

Test|Result|Duration|CPU Avg%|CPU Max%|RAM Avg MiB|RAM Max MiB|Sent Spans|Received Spans
----|------|-------:|-------:|-------:|----------:|----------:|---------:|-------------:
TestIdleMode|Passed|10s|0.3|1.0|25|34|0|0
Test10kSPS|Passed|17s|91.4|95.7|35|43|149610|149610
TestNoBackend10kSPS|Passed|10s|59.1|59.3|29|40|99910|0
Test1000SPSWithAttributes/0*0bytes|Passed|12s|18.7|19.3|30|41|10000|10000
Test1000SPSWithAttributes/100*50bytes|Passed|12s|70.2|71.0|32|45|10000|10000
Test1000SPSWithAttributes/10*1000bytes|Passed|12s|68.0|74.3|32|44|10000|10000
Test1000SPSWithAttributes/20*5000bytes|Passed|12s|158.2|173.7|39|55|10000|10000

Total duration: 87s
