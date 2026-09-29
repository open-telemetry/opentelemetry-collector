window.BENCHMARK_DATA = {
  "lastUpdate": 1790704121806,
  "repoUrl": "https://github.com/open-telemetry/opentelemetry-collector",
  "entries": {
    "Benchmark": [
      {
        "commit": {
          "author": {
            "email": "223565+codeboten@users.noreply.github.com",
            "name": "Alex Boten",
            "username": "codeboten"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "afdd9f6ba3994abd1d51f497126d54e1db4bddf7",
          "message": "[chore] fix path in download step (#16057)\n\n#### Description\n\nThe workflow was downloading the file to the wrong place. Attempting to\nfix\nhttps://github.com/open-telemetry/opentelemetry-collector/actions/runs/36568799205/job/109423816129\n\n#### Authorship\n\n- [x] I, a human, wrote this pull request description myself.\n\n<!--Please delete paragraphs that you did not use before submitting.-->\n\nSigned-off-by: Alex Boten <223565+codeboten@users.noreply.github.com>",
          "timestamp": "2026-09-29T16:43:14Z",
          "tree_id": "a669ffd6305556fd49b03d6947985491e7e69ac7",
          "url": "https://github.com/open-telemetry/opentelemetry-collector/commit/afdd9f6ba3994abd1d51f497126d54e1db4bddf7"
        },
        "date": 1790704107890,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkPersistentQueue",
            "value": 226551,
            "unit": "ns/op\t   98307 B/op\t    3131 allocs/op",
            "extra": "5362 times\n4 procs"
          },
          {
            "name": "BenchmarkPersistentQueue - ns/op",
            "value": 226551,
            "unit": "ns/op",
            "extra": "5362 times\n4 procs"
          },
          {
            "name": "BenchmarkPersistentQueue - B/op",
            "value": 98307,
            "unit": "B/op",
            "extra": "5362 times\n4 procs"
          },
          {
            "name": "BenchmarkPersistentQueue - allocs/op",
            "value": 3131,
            "unit": "allocs/op",
            "extra": "5362 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryQueueWaitForResult",
            "value": 84059,
            "unit": "ns/op\t    4800 B/op\t     100 allocs/op",
            "extra": "14328 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryQueueWaitForResult - ns/op",
            "value": 84059,
            "unit": "ns/op",
            "extra": "14328 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryQueueWaitForResult - B/op",
            "value": 4800,
            "unit": "B/op",
            "extra": "14328 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryQueueWaitForResult - allocs/op",
            "value": 100,
            "unit": "allocs/op",
            "extra": "14328 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeManySmallProfiles",
            "value": 5008580257,
            "unit": "ns/op\t4411228216 B/op\t110208124 allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeManySmallProfiles - ns/op",
            "value": 5008580257,
            "unit": "ns/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeManySmallProfiles - B/op",
            "value": 4411228216,
            "unit": "B/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeManySmallProfiles - allocs/op",
            "value": 110208124,
            "unit": "allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeManyProfilesSlightlyAboveLimit",
            "value": 116888006,
            "unit": "ns/op\t99206141 B/op\t 2101751 allocs/op",
            "extra": "9 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeManyProfilesSlightlyAboveLimit - ns/op",
            "value": 116888006,
            "unit": "ns/op",
            "extra": "9 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeManyProfilesSlightlyAboveLimit - B/op",
            "value": 99206141,
            "unit": "B/op",
            "extra": "9 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeManyProfilesSlightlyAboveLimit - allocs/op",
            "value": 2101751,
            "unit": "allocs/op",
            "extra": "9 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeHugeProfiles",
            "value": 105298996,
            "unit": "ns/op\t61237419 B/op\t 1190840 allocs/op",
            "extra": "10 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeHugeProfiles - ns/op",
            "value": 105298996,
            "unit": "ns/op",
            "extra": "10 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeHugeProfiles - B/op",
            "value": 61237419,
            "unit": "B/op",
            "extra": "10 times\n4 procs"
          },
          {
            "name": "BenchmarkSplittingBasedOnByteSizeHugeProfiles - allocs/op",
            "value": 1190840,
            "unit": "allocs/op",
            "extra": "10 times\n4 procs"
          },
          {
            "name": "BenchmarkAssertMutable",
            "value": 0.9361,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAssertMutable - ns/op",
            "value": 0.9361,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAssertMutable - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAssertMutable - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWriteInt64",
            "value": 34.26,
            "unit": "ns/op",
            "extra": "34745379 times\n4 procs"
          },
          {
            "name": "BenchmarkWriteUint64",
            "value": 33.38,
            "unit": "ns/op",
            "extra": "36098185 times\n4 procs"
          },
          {
            "name": "BenchmarkInt64SliceEqual",
            "value": 1.977,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "608189700 times\n4 procs"
          },
          {
            "name": "BenchmarkInt64SliceEqual - ns/op",
            "value": 1.977,
            "unit": "ns/op",
            "extra": "608189700 times\n4 procs"
          },
          {
            "name": "BenchmarkInt64SliceEqual - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "608189700 times\n4 procs"
          },
          {
            "name": "BenchmarkInt64SliceEqual - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "608189700 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/nil",
            "value": 3.115,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "385424805 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/nil - ns/op",
            "value": 3.115,
            "unit": "ns/op",
            "extra": "385424805 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/nil - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "385424805 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/nil - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "385424805 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/strings",
            "value": 8.406,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "142719470 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/strings - ns/op",
            "value": 8.406,
            "unit": "ns/op",
            "extra": "142719470 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/strings - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "142719470 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/strings - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "142719470 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/booleans",
            "value": 6.543,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "183346921 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/booleans - ns/op",
            "value": 6.543,
            "unit": "ns/op",
            "extra": "183346921 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/booleans - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "183346921 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/booleans - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "183346921 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/ints",
            "value": 5.617,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "214135870 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/ints - ns/op",
            "value": 5.617,
            "unit": "ns/op",
            "extra": "214135870 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/ints - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "214135870 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/ints - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "214135870 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/doubles",
            "value": 6.245,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "192216784 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/doubles - ns/op",
            "value": 6.245,
            "unit": "ns/op",
            "extra": "192216784 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/doubles - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "192216784 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/doubles - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "192216784 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/byte_slices",
            "value": 9.049,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "132629274 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/byte_slices - ns/op",
            "value": 9.049,
            "unit": "ns/op",
            "extra": "132629274 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/byte_slices - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "132629274 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/byte_slices - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "132629274 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/slices",
            "value": 19.04,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "60231375 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/slices - ns/op",
            "value": 19.04,
            "unit": "ns/op",
            "extra": "60231375 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/slices - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "60231375 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/slices - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "60231375 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/maps",
            "value": 26.55,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "44572947 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/maps - ns/op",
            "value": 26.55,
            "unit": "ns/op",
            "extra": "44572947 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/maps - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "44572947 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/maps - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "44572947 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/booleans",
            "value": 6.543,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "183346921 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/booleans - ns/op",
            "value": 6.543,
            "unit": "ns/op",
            "extra": "183346921 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/booleans - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "183346921 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/booleans - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "183346921 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/byte_slices",
            "value": 9.049,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "132629274 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/byte_slices - ns/op",
            "value": 9.049,
            "unit": "ns/op",
            "extra": "132629274 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/byte_slices - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "132629274 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/byte_slices - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "132629274 times\n4 procs"
          },
          {
            "name": "BenchmarkByteSliceEqual",
            "value": 2.493,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "480518182 times\n4 procs"
          },
          {
            "name": "BenchmarkByteSliceEqual - ns/op",
            "value": 2.493,
            "unit": "ns/op",
            "extra": "480518182 times\n4 procs"
          },
          {
            "name": "BenchmarkByteSliceEqual - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "480518182 times\n4 procs"
          },
          {
            "name": "BenchmarkByteSliceEqual - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "480518182 times\n4 procs"
          },
          {
            "name": "BenchmarkInt32SliceEqual",
            "value": 2.498,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "480974841 times\n4 procs"
          },
          {
            "name": "BenchmarkInt32SliceEqual - ns/op",
            "value": 2.498,
            "unit": "ns/op",
            "extra": "480974841 times\n4 procs"
          },
          {
            "name": "BenchmarkInt32SliceEqual - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "480974841 times\n4 procs"
          },
          {
            "name": "BenchmarkInt32SliceEqual - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "480974841 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/nil",
            "value": 3.115,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "385424805 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/nil - ns/op",
            "value": 3.115,
            "unit": "ns/op",
            "extra": "385424805 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/nil - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "385424805 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/nil - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "385424805 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/ints",
            "value": 5.617,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "214135870 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/ints - ns/op",
            "value": 5.617,
            "unit": "ns/op",
            "extra": "214135870 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/ints - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "214135870 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/ints - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "214135870 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/doubles",
            "value": 6.245,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "192216784 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/doubles - ns/op",
            "value": 6.245,
            "unit": "ns/op",
            "extra": "192216784 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/doubles - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "192216784 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/doubles - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "192216784 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/maps",
            "value": 26.55,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "44572947 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/maps - ns/op",
            "value": 26.55,
            "unit": "ns/op",
            "extra": "44572947 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/maps - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "44572947 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/maps - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "44572947 times\n4 procs"
          },
          {
            "name": "BenchmarkStringSliceEqual",
            "value": 7.987,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "147360510 times\n4 procs"
          },
          {
            "name": "BenchmarkStringSliceEqual - ns/op",
            "value": 7.987,
            "unit": "ns/op",
            "extra": "147360510 times\n4 procs"
          },
          {
            "name": "BenchmarkStringSliceEqual - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "147360510 times\n4 procs"
          },
          {
            "name": "BenchmarkStringSliceEqual - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "147360510 times\n4 procs"
          },
          {
            "name": "BenchmarkUInt64SliceEqual",
            "value": 2.805,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "427930483 times\n4 procs"
          },
          {
            "name": "BenchmarkUInt64SliceEqual - ns/op",
            "value": 2.805,
            "unit": "ns/op",
            "extra": "427930483 times\n4 procs"
          },
          {
            "name": "BenchmarkUInt64SliceEqual - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "427930483 times\n4 procs"
          },
          {
            "name": "BenchmarkUInt64SliceEqual - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "427930483 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/strings",
            "value": 8.406,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "142719470 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/strings - ns/op",
            "value": 8.406,
            "unit": "ns/op",
            "extra": "142719470 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/strings - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "142719470 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/strings - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "142719470 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/slices",
            "value": 19.04,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "60231375 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/slices - ns/op",
            "value": 19.04,
            "unit": "ns/op",
            "extra": "60231375 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/slices - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "60231375 times\n4 procs"
          },
          {
            "name": "BenchmarkValueEqual/slices - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "60231375 times\n4 procs"
          },
          {
            "name": "BenchmarkFloat64SliceEqual",
            "value": 2.214,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "541788094 times\n4 procs"
          },
          {
            "name": "BenchmarkFloat64SliceEqual - ns/op",
            "value": 2.214,
            "unit": "ns/op",
            "extra": "541788094 times\n4 procs"
          },
          {
            "name": "BenchmarkFloat64SliceEqual - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "541788094 times\n4 procs"
          },
          {
            "name": "BenchmarkFloat64SliceEqual - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "541788094 times\n4 procs"
          },
          {
            "name": "BenchmarkMapEqual",
            "value": 16.07,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "73167372 times\n4 procs"
          },
          {
            "name": "BenchmarkMapEqual - ns/op",
            "value": 16.07,
            "unit": "ns/op",
            "extra": "73167372 times\n4 procs"
          },
          {
            "name": "BenchmarkMapEqual - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "73167372 times\n4 procs"
          },
          {
            "name": "BenchmarkMapEqual - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "73167372 times\n4 procs"
          },
          {
            "name": "BenchmarkSliceEqual",
            "value": 11.92,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "99199369 times\n4 procs"
          },
          {
            "name": "BenchmarkSliceEqual - ns/op",
            "value": 11.92,
            "unit": "ns/op",
            "extra": "99199369 times\n4 procs"
          },
          {
            "name": "BenchmarkSliceEqual - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "99199369 times\n4 procs"
          },
          {
            "name": "BenchmarkSliceEqual - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "99199369 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsToProto2k",
            "value": 62691,
            "unit": "ns/op",
            "extra": "19167 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsFromProto2k",
            "value": 199436,
            "unit": "ns/op\t  304537 B/op\t    2019 allocs/op",
            "extra": "6010 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsFromProto2k - ns/op",
            "value": 199436,
            "unit": "ns/op",
            "extra": "6010 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsFromProto2k - B/op",
            "value": 304537,
            "unit": "B/op",
            "extra": "6010 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsFromProto2k - allocs/op",
            "value": 2019,
            "unit": "allocs/op",
            "extra": "6010 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsUsage",
            "value": 3143,
            "unit": "ns/op\t     672 B/op\t      26 allocs/op",
            "extra": "382137 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsUsage - ns/op",
            "value": 3143,
            "unit": "ns/op",
            "extra": "382137 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsUsage - B/op",
            "value": 672,
            "unit": "B/op",
            "extra": "382137 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsUsage - allocs/op",
            "value": 26,
            "unit": "allocs/op",
            "extra": "382137 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsMarshalJSON",
            "value": 4761,
            "unit": "ns/op\t    1888 B/op\t       3 allocs/op",
            "extra": "248539 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsMarshalJSON - ns/op",
            "value": 4761,
            "unit": "ns/op",
            "extra": "248539 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsMarshalJSON - B/op",
            "value": 1888,
            "unit": "B/op",
            "extra": "248539 times\n4 procs"
          },
          {
            "name": "BenchmarkLogsMarshalJSON - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "248539 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsToProto2k",
            "value": 115097,
            "unit": "ns/op",
            "extra": "9476 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsFromProto10k",
            "value": 635211,
            "unit": "ns/op\t  576541 B/op\t   14019 allocs/op",
            "extra": "1868 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsFromProto10k - ns/op",
            "value": 635211,
            "unit": "ns/op",
            "extra": "1868 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsFromProto10k - B/op",
            "value": 576541,
            "unit": "B/op",
            "extra": "1868 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsFromProto10k - allocs/op",
            "value": 14019,
            "unit": "allocs/op",
            "extra": "1868 times\n4 procs"
          },
          {
            "name": "BenchmarkOtlpToFromInternal_PassThrough",
            "value": 1.431,
            "unit": "ns/op",
            "extra": "840352405 times\n4 procs"
          },
          {
            "name": "BenchmarkOtlpToFromInternal_Gauge_MutateOneLabel",
            "value": 47.95,
            "unit": "ns/op",
            "extra": "24719510 times\n4 procs"
          },
          {
            "name": "BenchmarkOtlpToFromInternal_Sum_MutateOneLabel",
            "value": 47.79,
            "unit": "ns/op",
            "extra": "24924373 times\n4 procs"
          },
          {
            "name": "BenchmarkOtlpToFromInternal_HistogramPoints_MutateOneLabel",
            "value": 51.64,
            "unit": "ns/op",
            "extra": "24118567 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsUsage",
            "value": 1616,
            "unit": "ns/op\t     160 B/op\t      10 allocs/op",
            "extra": "729032 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsUsage - ns/op",
            "value": 1616,
            "unit": "ns/op",
            "extra": "729032 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsUsage - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "729032 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsUsage - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "729032 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsMarshalJSON",
            "value": 6359,
            "unit": "ns/op\t    2401 B/op\t       3 allocs/op",
            "extra": "187093 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsMarshalJSON - ns/op",
            "value": 6359,
            "unit": "ns/op",
            "extra": "187093 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsMarshalJSON - B/op",
            "value": 2401,
            "unit": "B/op",
            "extra": "187093 times\n4 procs"
          },
          {
            "name": "BenchmarkMetricsMarshalJSON - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "187093 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesFromProto2k",
            "value": 384336,
            "unit": "ns/op\t  528539 B/op\t    4019 allocs/op",
            "extra": "3309 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesFromProto2k - ns/op",
            "value": 384336,
            "unit": "ns/op",
            "extra": "3309 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesFromProto2k - B/op",
            "value": 528539,
            "unit": "B/op",
            "extra": "3309 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesFromProto2k - allocs/op",
            "value": 4019,
            "unit": "allocs/op",
            "extra": "3309 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesUsage",
            "value": 5533,
            "unit": "ns/op\t    1072 B/op\t      34 allocs/op",
            "extra": "214939 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesUsage - ns/op",
            "value": 5533,
            "unit": "ns/op",
            "extra": "214939 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesUsage - B/op",
            "value": 1072,
            "unit": "B/op",
            "extra": "214939 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesUsage - allocs/op",
            "value": 34,
            "unit": "allocs/op",
            "extra": "214939 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesMarshalJSON",
            "value": 7313,
            "unit": "ns/op\t    3169 B/op\t       3 allocs/op",
            "extra": "159957 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesMarshalJSON - ns/op",
            "value": 7313,
            "unit": "ns/op",
            "extra": "159957 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesMarshalJSON - B/op",
            "value": 3169,
            "unit": "B/op",
            "extra": "159957 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesMarshalJSON - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "159957 times\n4 procs"
          },
          {
            "name": "BenchmarkTracesToProto2k",
            "value": 101443,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesMarshalJSON",
            "value": 5316,
            "unit": "ns/op\t    1892 B/op\t       4 allocs/op",
            "extra": "223881 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesMarshalJSON - ns/op",
            "value": 5316,
            "unit": "ns/op",
            "extra": "223881 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesMarshalJSON - B/op",
            "value": 1892,
            "unit": "B/op",
            "extra": "223881 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesMarshalJSON - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "223881 times\n4 procs"
          },
          {
            "name": "BenchmarkKeyValueAndUnitSwitchDictionary",
            "value": 120,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "9920690 times\n4 procs"
          },
          {
            "name": "BenchmarkKeyValueAndUnitSwitchDictionary - ns/op",
            "value": 120,
            "unit": "ns/op",
            "extra": "9920690 times\n4 procs"
          },
          {
            "name": "BenchmarkKeyValueAndUnitSwitchDictionary - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "9920690 times\n4 procs"
          },
          {
            "name": "BenchmarkKeyValueAndUnitSwitchDictionary - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "9920690 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_an_existing_location",
            "value": 9.994,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120045038 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_an_existing_location - ns/op",
            "value": 9.994,
            "unit": "ns/op",
            "extra": "120045038 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_an_existing_location - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120045038 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_an_existing_location - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120045038 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_duplicate_attribute",
            "value": 9.968,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120415623 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_duplicate_attribute - ns/op",
            "value": 9.968,
            "unit": "ns/op",
            "extra": "120415623 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_duplicate_attribute - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120415623 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_duplicate_attribute - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120415623 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=20",
            "value": 81302209,
            "unit": "ns/op\t17468435 B/op\t  194252 allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=20 - ns/op",
            "value": 81302209,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=20 - B/op",
            "value": 17468435,
            "unit": "B/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=20 - allocs/op",
            "value": 194252,
            "unit": "allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_new_link",
            "value": 5.627,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "213046438 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_new_link - ns/op",
            "value": 5.627,
            "unit": "ns/op",
            "extra": "213046438 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_new_link - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "213046438 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_new_link - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "213046438 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_an_existing_link",
            "value": 5.613,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "213822750 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_an_existing_link - ns/op",
            "value": 5.613,
            "unit": "ns/op",
            "extra": "213822750 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_an_existing_link - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "213822750 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_an_existing_link - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "213822750 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_duplicate_link",
            "value": 5.632,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "213629244 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_duplicate_link - ns/op",
            "value": 5.632,
            "unit": "ns/op",
            "extra": "213629244 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_duplicate_link - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "213629244 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_duplicate_link - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "213629244 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_hundred_links_to_loop_through",
            "value": 5.616,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "213710612 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_hundred_links_to_loop_through - ns/op",
            "value": 5.616,
            "unit": "ns/op",
            "extra": "213710612 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_hundred_links_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "213710612 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_hundred_links_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "213710612 times\n4 procs"
          },
          {
            "name": "BenchmarkFromLocationIndices",
            "value": 2640,
            "unit": "ns/op\t    3648 B/op\t      53 allocs/op",
            "extra": "418508 times\n4 procs"
          },
          {
            "name": "BenchmarkFromLocationIndices - ns/op",
            "value": 2640,
            "unit": "ns/op",
            "extra": "418508 times\n4 procs"
          },
          {
            "name": "BenchmarkFromLocationIndices - B/op",
            "value": 3648,
            "unit": "B/op",
            "extra": "418508 times\n4 procs"
          },
          {
            "name": "BenchmarkFromLocationIndices - allocs/op",
            "value": 53,
            "unit": "allocs/op",
            "extra": "418508 times\n4 procs"
          },
          {
            "name": "BenchmarkMappingSwitchDictionary",
            "value": 1058,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMappingSwitchDictionary - ns/op",
            "value": 1058,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMappingSwitchDictionary - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMappingSwitchDictionary - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_new_mapping",
            "value": 8.097,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "148201236 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_new_mapping - ns/op",
            "value": 8.097,
            "unit": "ns/op",
            "extra": "148201236 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_new_mapping - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "148201236 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_new_mapping - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "148201236 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_an_existing_mapping",
            "value": 8.099,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "148134513 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_an_existing_mapping - ns/op",
            "value": 8.099,
            "unit": "ns/op",
            "extra": "148134513 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_an_existing_mapping - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "148134513 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_an_existing_mapping - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "148134513 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_duplicate_mapping",
            "value": 8.099,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "148144936 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_duplicate_mapping - ns/op",
            "value": 8.099,
            "unit": "ns/op",
            "extra": "148144936 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_duplicate_mapping - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "148144936 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_duplicate_mapping - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "148144936 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_hundred_mappings_to_loop_through",
            "value": 11.25,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_hundred_mappings_to_loop_through - ns/op",
            "value": 11.25,
            "unit": "ns/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_hundred_mappings_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_hundred_mappings_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/large",
            "value": 35245075,
            "unit": "ns/op\t36549217 B/op\t  604598 allocs/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/large - ns/op",
            "value": 35245075,
            "unit": "ns/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/large - B/op",
            "value": 36549217,
            "unit": "B/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/large - allocs/op",
            "value": 604598,
            "unit": "allocs/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_new_function",
            "value": 4.991,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "240069716 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_new_function - ns/op",
            "value": 4.991,
            "unit": "ns/op",
            "extra": "240069716 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_new_function - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "240069716 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_new_function - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "240069716 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/small",
            "value": 13215,
            "unit": "ns/op\t   15232 B/op\t     258 allocs/op",
            "extra": "88669 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/small - ns/op",
            "value": 13215,
            "unit": "ns/op",
            "extra": "88669 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/small - B/op",
            "value": 15232,
            "unit": "B/op",
            "extra": "88669 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/small - allocs/op",
            "value": 258,
            "unit": "allocs/op",
            "extra": "88669 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/without_refs",
            "value": 7162,
            "unit": "ns/op\t    3584 B/op\t      16 allocs/op",
            "extra": "219429 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/without_refs - ns/op",
            "value": 7162,
            "unit": "ns/op",
            "extra": "219429 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/without_refs - B/op",
            "value": 3584,
            "unit": "B/op",
            "extra": "219429 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/without_refs - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "219429 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/without_refs",
            "value": 266590,
            "unit": "ns/op\t   79232 B/op\t      43 allocs/op",
            "extra": "4496 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/without_refs - ns/op",
            "value": 266590,
            "unit": "ns/op",
            "extra": "4496 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/without_refs - B/op",
            "value": 79232,
            "unit": "B/op",
            "extra": "4496 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/without_refs - allocs/op",
            "value": 43,
            "unit": "allocs/op",
            "extra": "4496 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/without_refs",
            "value": 8014749,
            "unit": "ns/op\t 2125192 B/op\t     136 allocs/op",
            "extra": "147 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/without_refs - ns/op",
            "value": 8014749,
            "unit": "ns/op",
            "extra": "147 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/without_refs - B/op",
            "value": 2125192,
            "unit": "B/op",
            "extra": "147 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/without_refs - allocs/op",
            "value": 136,
            "unit": "allocs/op",
            "extra": "147 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_duplicate_stack",
            "value": 6.865,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "174869004 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_duplicate_stack - ns/op",
            "value": 6.865,
            "unit": "ns/op",
            "extra": "174869004 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_duplicate_stack - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "174869004 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_duplicate_stack - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "174869004 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=50",
            "value": 230400642,
            "unit": "ns/op\t71954499 B/op\t  772821 allocs/op",
            "extra": "5 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=50 - ns/op",
            "value": 230400642,
            "unit": "ns/op",
            "extra": "5 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=50 - B/op",
            "value": 71954499,
            "unit": "B/op",
            "extra": "5 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=50 - allocs/op",
            "value": 772821,
            "unit": "allocs/op",
            "extra": "5 times\n4 procs"
          },
          {
            "name": "BenchmarkLineSwitchDictionary",
            "value": 123.5,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "9654139 times\n4 procs"
          },
          {
            "name": "BenchmarkLineSwitchDictionary - ns/op",
            "value": 123.5,
            "unit": "ns/op",
            "extra": "9654139 times\n4 procs"
          },
          {
            "name": "BenchmarkLineSwitchDictionary - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "9654139 times\n4 procs"
          },
          {
            "name": "BenchmarkLineSwitchDictionary - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "9654139 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_duplicate_link",
            "value": 5.632,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "213629244 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_duplicate_link - ns/op",
            "value": 5.632,
            "unit": "ns/op",
            "extra": "213629244 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_duplicate_link - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "213629244 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_duplicate_link - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "213629244 times\n4 procs"
          },
          {
            "name": "BenchmarkResourceProfilesSwitchDictionary",
            "value": 263.4,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "4605897 times\n4 procs"
          },
          {
            "name": "BenchmarkResourceProfilesSwitchDictionary - ns/op",
            "value": 263.4,
            "unit": "ns/op",
            "extra": "4605897 times\n4 procs"
          },
          {
            "name": "BenchmarkResourceProfilesSwitchDictionary - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "4605897 times\n4 procs"
          },
          {
            "name": "BenchmarkResourceProfilesSwitchDictionary - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "4605897 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_hundred_values_to_loop_through",
            "value": 107.7,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "11037367 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_hundred_values_to_loop_through - ns/op",
            "value": 107.7,
            "unit": "ns/op",
            "extra": "11037367 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_hundred_values_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "11037367 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_hundred_values_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "11037367 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_hundred_locations_to_loop_through",
            "value": 13.4,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "88760232 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_hundred_locations_to_loop_through - ns/op",
            "value": 13.4,
            "unit": "ns/op",
            "extra": "88760232 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_hundred_locations_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "88760232 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_hundred_locations_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "88760232 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=1x",
            "value": 80044476,
            "unit": "ns/op\t17491859 B/op\t  194253 allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=1x - ns/op",
            "value": 80044476,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=1x - B/op",
            "value": 17491859,
            "unit": "B/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=1x - allocs/op",
            "value": 194253,
            "unit": "allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_hundred_links_to_loop_through",
            "value": 5.616,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "213710612 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_hundred_links_to_loop_through - ns/op",
            "value": 5.616,
            "unit": "ns/op",
            "extra": "213710612 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_hundred_links_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "213710612 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_hundred_links_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "213710612 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_new_stack",
            "value": 6.855,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "175106401 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_new_stack - ns/op",
            "value": 6.855,
            "unit": "ns/op",
            "extra": "175106401 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_new_stack - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "175106401 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_new_stack - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "175106401 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=0%",
            "value": 197350129,
            "unit": "ns/op\t105595432 B/op\t 1103129 allocs/op",
            "extra": "6 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=0% - ns/op",
            "value": 197350129,
            "unit": "ns/op",
            "extra": "6 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=0% - B/op",
            "value": 105595432,
            "unit": "B/op",
            "extra": "6 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=0% - allocs/op",
            "value": 1103129,
            "unit": "allocs/op",
            "extra": "6 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=50%",
            "value": 130813063,
            "unit": "ns/op\t55986452 B/op\t  593505 allocs/op",
            "extra": "8 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=50% - ns/op",
            "value": 130813063,
            "unit": "ns/op",
            "extra": "8 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=50% - B/op",
            "value": 55986452,
            "unit": "B/op",
            "extra": "8 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=50% - allocs/op",
            "value": 593505,
            "unit": "allocs/op",
            "extra": "8 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=90%",
            "value": 79870994,
            "unit": "ns/op\t17487956 B/op\t  194254 allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=90% - ns/op",
            "value": 79870994,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=90% - B/op",
            "value": 17487956,
            "unit": "B/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=90% - allocs/op",
            "value": 194254,
            "unit": "allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=99%",
            "value": 72656287,
            "unit": "ns/op\t10628252 B/op\t  104656 allocs/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=99% - ns/op",
            "value": 72656287,
            "unit": "ns/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=99% - B/op",
            "value": 10628252,
            "unit": "B/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=99% - allocs/op",
            "value": 104656,
            "unit": "allocs/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=5",
            "value": 19740904,
            "unit": "ns/op\t 3598487 B/op\t   41639 allocs/op",
            "extra": "60 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=5 - ns/op",
            "value": 19740904,
            "unit": "ns/op",
            "extra": "60 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=5 - B/op",
            "value": 3598487,
            "unit": "B/op",
            "extra": "60 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=5 - allocs/op",
            "value": 41639,
            "unit": "allocs/op",
            "extra": "60 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=20",
            "value": 81302209,
            "unit": "ns/op\t17468435 B/op\t  194252 allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=20 - ns/op",
            "value": 81302209,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=20 - B/op",
            "value": 17468435,
            "unit": "B/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=20 - allocs/op",
            "value": 194252,
            "unit": "allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=50",
            "value": 230400642,
            "unit": "ns/op\t71954499 B/op\t  772821 allocs/op",
            "extra": "5 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=50 - ns/op",
            "value": 230400642,
            "unit": "ns/op",
            "extra": "5 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=50 - B/op",
            "value": 71954499,
            "unit": "B/op",
            "extra": "5 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=50 - allocs/op",
            "value": 772821,
            "unit": "allocs/op",
            "extra": "5 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=0.5x",
            "value": 40446047,
            "unit": "ns/op\t 8732142 B/op\t   97400 allocs/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=0.5x - ns/op",
            "value": 40446047,
            "unit": "ns/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=0.5x - B/op",
            "value": 8732142,
            "unit": "B/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=0.5x - allocs/op",
            "value": 97400,
            "unit": "allocs/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=1x",
            "value": 80044476,
            "unit": "ns/op\t17491859 B/op\t  194253 allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=1x - ns/op",
            "value": 80044476,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=1x - B/op",
            "value": 17491859,
            "unit": "B/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=1x - allocs/op",
            "value": 194253,
            "unit": "allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=2x",
            "value": 163465977,
            "unit": "ns/op\t35388281 B/op\t  388156 allocs/op",
            "extra": "7 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=2x - ns/op",
            "value": 163465977,
            "unit": "ns/op",
            "extra": "7 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=2x - B/op",
            "value": 35388281,
            "unit": "B/op",
            "extra": "7 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=2x - allocs/op",
            "value": 388156,
            "unit": "allocs/op",
            "extra": "7 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=0%",
            "value": 197350129,
            "unit": "ns/op\t105595432 B/op\t 1103129 allocs/op",
            "extra": "6 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=0% - ns/op",
            "value": 197350129,
            "unit": "ns/op",
            "extra": "6 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=0% - B/op",
            "value": 105595432,
            "unit": "B/op",
            "extra": "6 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=0% - allocs/op",
            "value": 1103129,
            "unit": "allocs/op",
            "extra": "6 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=90%",
            "value": 79870994,
            "unit": "ns/op\t17487956 B/op\t  194254 allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=90% - ns/op",
            "value": 79870994,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=90% - B/op",
            "value": 17487956,
            "unit": "B/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=90% - allocs/op",
            "value": 194254,
            "unit": "allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=2x",
            "value": 163465977,
            "unit": "ns/op\t35388281 B/op\t  388156 allocs/op",
            "extra": "7 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=2x - ns/op",
            "value": 163465977,
            "unit": "ns/op",
            "extra": "7 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=2x - B/op",
            "value": 35388281,
            "unit": "B/op",
            "extra": "7 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=2x - allocs/op",
            "value": 388156,
            "unit": "allocs/op",
            "extra": "7 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=8000",
            "value": 23056760,
            "unit": "ns/op\t10875946 B/op\t  146960 allocs/op",
            "extra": "52 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=8000 - ns/op",
            "value": 23056760,
            "unit": "ns/op",
            "extra": "52 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=8000 - B/op",
            "value": 10875946,
            "unit": "B/op",
            "extra": "52 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=8000 - allocs/op",
            "value": 146960,
            "unit": "allocs/op",
            "extra": "52 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_new_location",
            "value": 9.971,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120319208 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_new_location - ns/op",
            "value": 9.971,
            "unit": "ns/op",
            "extra": "120319208 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_new_location - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120319208 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_new_location - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120319208 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_an_existing_location",
            "value": 9.994,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120045038 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_an_existing_location - ns/op",
            "value": 9.994,
            "unit": "ns/op",
            "extra": "120045038 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_an_existing_location - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120045038 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_an_existing_location - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120045038 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_duplicate_location",
            "value": 9.988,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120140496 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_duplicate_location - ns/op",
            "value": 9.988,
            "unit": "ns/op",
            "extra": "120140496 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_duplicate_location - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120140496 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_duplicate_location - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120140496 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_hundred_locations_to_loop_through",
            "value": 394,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "3057266 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_hundred_locations_to_loop_through - ns/op",
            "value": 394,
            "unit": "ns/op",
            "extra": "3057266 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_hundred_locations_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "3057266 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_hundred_locations_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "3057266 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=500",
            "value": 1091918,
            "unit": "ns/op\t  697536 B/op\t    9303 allocs/op",
            "extra": "1090 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=500 - ns/op",
            "value": 1091918,
            "unit": "ns/op",
            "extra": "1090 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=500 - B/op",
            "value": 697536,
            "unit": "B/op",
            "extra": "1090 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=500 - allocs/op",
            "value": 9303,
            "unit": "allocs/op",
            "extra": "1090 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_an_existing_function",
            "value": 4.994,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "240255871 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_an_existing_function - ns/op",
            "value": 4.994,
            "unit": "ns/op",
            "extra": "240255871 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_an_existing_function - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "240255871 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_an_existing_function - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "240255871 times\n4 procs"
          },
          {
            "name": "BenchmarkSampleSwitchDictionary",
            "value": 4.669,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "257022446 times\n4 procs"
          },
          {
            "name": "BenchmarkSampleSwitchDictionary - ns/op",
            "value": 4.669,
            "unit": "ns/op",
            "extra": "257022446 times\n4 procs"
          },
          {
            "name": "BenchmarkSampleSwitchDictionary - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "257022446 times\n4 procs"
          },
          {
            "name": "BenchmarkSampleSwitchDictionary - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "257022446 times\n4 procs"
          },
          {
            "name": "BenchmarkStackSwitchDictionary",
            "value": 1179,
            "unit": "ns/op\t     112 B/op\t       3 allocs/op",
            "extra": "989938 times\n4 procs"
          },
          {
            "name": "BenchmarkStackSwitchDictionary - ns/op",
            "value": 1179,
            "unit": "ns/op",
            "extra": "989938 times\n4 procs"
          },
          {
            "name": "BenchmarkStackSwitchDictionary - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "989938 times\n4 procs"
          },
          {
            "name": "BenchmarkStackSwitchDictionary - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "989938 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_new_stack",
            "value": 6.855,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "175106401 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_new_stack - ns/op",
            "value": 6.855,
            "unit": "ns/op",
            "extra": "175106401 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_new_stack - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "175106401 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_new_stack - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "175106401 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_an_existing_stack",
            "value": 7.176,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "167176219 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_an_existing_stack - ns/op",
            "value": 7.176,
            "unit": "ns/op",
            "extra": "167176219 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_an_existing_stack - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "167176219 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_an_existing_stack - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "167176219 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_duplicate_stack",
            "value": 6.865,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "174869004 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_duplicate_stack - ns/op",
            "value": 6.865,
            "unit": "ns/op",
            "extra": "174869004 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_duplicate_stack - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "174869004 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_duplicate_stack - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "174869004 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_hundred_stacks_to_loop_through",
            "value": 333.6,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "3582045 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_hundred_stacks_to_loop_through - ns/op",
            "value": 333.6,
            "unit": "ns/op",
            "extra": "3582045 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_hundred_stacks_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "3582045 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_hundred_stacks_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "3582045 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_an_existing_stack",
            "value": 7.176,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "167176219 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_an_existing_stack - ns/op",
            "value": 7.176,
            "unit": "ns/op",
            "extra": "167176219 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_an_existing_stack - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "167176219 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_an_existing_stack - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "167176219 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_new_value",
            "value": 6.236,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "192443348 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_new_value - ns/op",
            "value": 6.236,
            "unit": "ns/op",
            "extra": "192443348 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_new_value - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "192443348 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_new_value - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "192443348 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_an_existing_value",
            "value": 6.237,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "192481712 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_an_existing_value - ns/op",
            "value": 6.237,
            "unit": "ns/op",
            "extra": "192481712 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_an_existing_value - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "192481712 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_an_existing_value - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "192481712 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_duplicate_value",
            "value": 6.259,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "192504602 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_duplicate_value - ns/op",
            "value": 6.259,
            "unit": "ns/op",
            "extra": "192504602 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_duplicate_value - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "192504602 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_duplicate_value - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "192504602 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_hundred_values_to_loop_through",
            "value": 107.7,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "11037367 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_hundred_values_to_loop_through - ns/op",
            "value": 107.7,
            "unit": "ns/op",
            "extra": "11037367 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_hundred_values_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "11037367 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_hundred_values_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "11037367 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_new_value",
            "value": 6.236,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "192443348 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_new_value - ns/op",
            "value": 6.236,
            "unit": "ns/op",
            "extra": "192443348 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_new_value - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "192443348 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_new_value - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "192443348 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_new_link",
            "value": 5.627,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "213046438 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_new_link - ns/op",
            "value": 5.627,
            "unit": "ns/op",
            "extra": "213046438 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_new_link - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "213046438 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_a_new_link - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "213046438 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_an_existing_mapping",
            "value": 8.099,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "148134513 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_an_existing_mapping - ns/op",
            "value": 8.099,
            "unit": "ns/op",
            "extra": "148134513 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_an_existing_mapping - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "148134513 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_an_existing_mapping - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "148134513 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/with_refs",
            "value": 3225,
            "unit": "ns/op\t    1152 B/op\t       1 allocs/op",
            "extra": "369034 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/with_refs - ns/op",
            "value": 3225,
            "unit": "ns/op",
            "extra": "369034 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/with_refs - B/op",
            "value": 1152,
            "unit": "B/op",
            "extra": "369034 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/with_refs - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "369034 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/without_refs",
            "value": 7162,
            "unit": "ns/op\t    3584 B/op\t      16 allocs/op",
            "extra": "219429 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/without_refs - ns/op",
            "value": 7162,
            "unit": "ns/op",
            "extra": "219429 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/without_refs - B/op",
            "value": 3584,
            "unit": "B/op",
            "extra": "219429 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/without_refs - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "219429 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/with_refs",
            "value": 263122,
            "unit": "ns/op\t   73728 B/op\t       1 allocs/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/with_refs - ns/op",
            "value": 263122,
            "unit": "ns/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/with_refs - B/op",
            "value": 73728,
            "unit": "B/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/with_refs - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/without_refs",
            "value": 266590,
            "unit": "ns/op\t   79232 B/op\t      43 allocs/op",
            "extra": "4496 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/without_refs - ns/op",
            "value": 266590,
            "unit": "ns/op",
            "extra": "4496 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/without_refs - B/op",
            "value": 79232,
            "unit": "B/op",
            "extra": "4496 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/without_refs - allocs/op",
            "value": 43,
            "unit": "allocs/op",
            "extra": "4496 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/with_refs",
            "value": 8334807,
            "unit": "ns/op\t 2113563 B/op\t       1 allocs/op",
            "extra": "144 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/with_refs - ns/op",
            "value": 8334807,
            "unit": "ns/op",
            "extra": "144 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/with_refs - B/op",
            "value": 2113563,
            "unit": "B/op",
            "extra": "144 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/with_refs - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "144 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/without_refs",
            "value": 8014749,
            "unit": "ns/op\t 2125192 B/op\t     136 allocs/op",
            "extra": "147 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/without_refs - ns/op",
            "value": 8014749,
            "unit": "ns/op",
            "extra": "147 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/without_refs - B/op",
            "value": 2125192,
            "unit": "B/op",
            "extra": "147 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/without_refs - allocs/op",
            "value": 136,
            "unit": "allocs/op",
            "extra": "147 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/with_refs",
            "value": 263122,
            "unit": "ns/op\t   73728 B/op\t       1 allocs/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/with_refs - ns/op",
            "value": 263122,
            "unit": "ns/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/with_refs - B/op",
            "value": 73728,
            "unit": "B/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/medium/with_refs - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=5",
            "value": 19740904,
            "unit": "ns/op\t 3598487 B/op\t   41639 allocs/op",
            "extra": "60 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=5 - ns/op",
            "value": 19740904,
            "unit": "ns/op",
            "extra": "60 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=5 - B/op",
            "value": 3598487,
            "unit": "B/op",
            "extra": "60 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/batch/batch=5 - allocs/op",
            "value": 41639,
            "unit": "allocs/op",
            "extra": "60 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=0.5x",
            "value": 40446047,
            "unit": "ns/op\t 8732142 B/op\t   97400 allocs/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=0.5x - ns/op",
            "value": 40446047,
            "unit": "ns/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=0.5x - B/op",
            "value": 8732142,
            "unit": "B/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/size/scale=0.5x - allocs/op",
            "value": 97400,
            "unit": "allocs/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkFunctionSwitchDictionary",
            "value": 776.7,
            "unit": "ns/op\t      64 B/op\t       1 allocs/op",
            "extra": "1553653 times\n4 procs"
          },
          {
            "name": "BenchmarkFunctionSwitchDictionary - ns/op",
            "value": 776.7,
            "unit": "ns/op",
            "extra": "1553653 times\n4 procs"
          },
          {
            "name": "BenchmarkFunctionSwitchDictionary - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "1553653 times\n4 procs"
          },
          {
            "name": "BenchmarkFunctionSwitchDictionary - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "1553653 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_hundred_functions_to_loop_through",
            "value": 6.076,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "197334309 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_hundred_functions_to_loop_through - ns/op",
            "value": 6.076,
            "unit": "ns/op",
            "extra": "197334309 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_hundred_functions_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "197334309 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_hundred_functions_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "197334309 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_hundred_mappings_to_loop_through",
            "value": 11.25,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_hundred_mappings_to_loop_through - ns/op",
            "value": 11.25,
            "unit": "ns/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_hundred_mappings_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_hundred_mappings_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesFromProto",
            "value": 9205,
            "unit": "ns/op\t   15048 B/op\t     144 allocs/op",
            "extra": "129114 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesFromProto - ns/op",
            "value": 9205,
            "unit": "ns/op",
            "extra": "129114 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesFromProto - B/op",
            "value": 15048,
            "unit": "B/op",
            "extra": "129114 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesFromProto - allocs/op",
            "value": 144,
            "unit": "allocs/op",
            "extra": "129114 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/with_refs",
            "value": 8334807,
            "unit": "ns/op\t 2113563 B/op\t       1 allocs/op",
            "extra": "144 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/with_refs - ns/op",
            "value": 8334807,
            "unit": "ns/op",
            "extra": "144 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/with_refs - B/op",
            "value": 2113563,
            "unit": "B/op",
            "extra": "144 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/large/with_refs - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "144 times\n4 procs"
          },
          {
            "name": "BenchmarkProfileSwitchDictionary",
            "value": 1031,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfileSwitchDictionary - ns/op",
            "value": 1031,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfileSwitchDictionary - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfileSwitchDictionary - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=99%",
            "value": 72656287,
            "unit": "ns/op\t10628252 B/op\t  104656 allocs/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=99% - ns/op",
            "value": 72656287,
            "unit": "ns/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=99% - B/op",
            "value": 10628252,
            "unit": "B/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=99% - allocs/op",
            "value": 104656,
            "unit": "allocs/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_an_existing_attribute",
            "value": 9.98,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120230799 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_an_existing_attribute - ns/op",
            "value": 9.98,
            "unit": "ns/op",
            "extra": "120230799 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_an_existing_attribute - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120230799 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_an_existing_attribute - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120230799 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=500",
            "value": 1091918,
            "unit": "ns/op\t  697536 B/op\t    9303 allocs/op",
            "extra": "1090 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=500 - ns/op",
            "value": 1091918,
            "unit": "ns/op",
            "extra": "1090 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=500 - B/op",
            "value": 697536,
            "unit": "B/op",
            "extra": "1090 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=500 - allocs/op",
            "value": 9303,
            "unit": "allocs/op",
            "extra": "1090 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=2000",
            "value": 5596756,
            "unit": "ns/op\t 2678586 B/op\t   36836 allocs/op",
            "extra": "214 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=2000 - ns/op",
            "value": 5596756,
            "unit": "ns/op",
            "extra": "214 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=2000 - B/op",
            "value": 2678586,
            "unit": "B/op",
            "extra": "214 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=2000 - allocs/op",
            "value": 36836,
            "unit": "allocs/op",
            "extra": "214 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=8000",
            "value": 23056760,
            "unit": "ns/op\t10875946 B/op\t  146960 allocs/op",
            "extra": "52 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=8000 - ns/op",
            "value": 23056760,
            "unit": "ns/op",
            "extra": "52 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=8000 - B/op",
            "value": 10875946,
            "unit": "B/op",
            "extra": "52 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=8000 - allocs/op",
            "value": 146960,
            "unit": "allocs/op",
            "extra": "52 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_new_mapping",
            "value": 8.097,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "148201236 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_new_mapping - ns/op",
            "value": 8.097,
            "unit": "ns/op",
            "extra": "148201236 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_new_mapping - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "148201236 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_new_mapping - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "148201236 times\n4 procs"
          },
          {
            "name": "BenchmarkFromAttributeIndices",
            "value": 255.8,
            "unit": "ns/op\t     224 B/op\t       7 allocs/op",
            "extra": "4641248 times\n4 procs"
          },
          {
            "name": "BenchmarkFromAttributeIndices - ns/op",
            "value": 255.8,
            "unit": "ns/op",
            "extra": "4641248 times\n4 procs"
          },
          {
            "name": "BenchmarkFromAttributeIndices - B/op",
            "value": 224,
            "unit": "B/op",
            "extra": "4641248 times\n4 procs"
          },
          {
            "name": "BenchmarkFromAttributeIndices - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "4641248 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_duplicate_mapping",
            "value": 8.099,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "148144936 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_duplicate_mapping - ns/op",
            "value": 8.099,
            "unit": "ns/op",
            "extra": "148144936 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_duplicate_mapping - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "148144936 times\n4 procs"
          },
          {
            "name": "BenchmarkSetMapping/with_a_duplicate_mapping - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "148144936 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesUsage",
            "value": 3918,
            "unit": "ns/op\t     624 B/op\t      26 allocs/op",
            "extra": "291592 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesUsage - ns/op",
            "value": 3918,
            "unit": "ns/op",
            "extra": "291592 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesUsage - B/op",
            "value": 624,
            "unit": "B/op",
            "extra": "291592 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesUsage - allocs/op",
            "value": 26,
            "unit": "allocs/op",
            "extra": "291592 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_an_existing_link",
            "value": 5.613,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "213822750 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_an_existing_link - ns/op",
            "value": 5.613,
            "unit": "ns/op",
            "extra": "213822750 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_an_existing_link - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "213822750 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLink/with_an_existing_link - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "213822750 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/small",
            "value": 13215,
            "unit": "ns/op\t   15232 B/op\t     258 allocs/op",
            "extra": "88669 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/small - ns/op",
            "value": 13215,
            "unit": "ns/op",
            "extra": "88669 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/small - B/op",
            "value": 15232,
            "unit": "B/op",
            "extra": "88669 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/small - allocs/op",
            "value": 258,
            "unit": "allocs/op",
            "extra": "88669 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/medium",
            "value": 1170744,
            "unit": "ns/op\t 1239317 B/op\t   20425 allocs/op",
            "extra": "1008 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/medium - ns/op",
            "value": 1170744,
            "unit": "ns/op",
            "extra": "1008 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/medium - B/op",
            "value": 1239317,
            "unit": "B/op",
            "extra": "1008 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/medium - allocs/op",
            "value": 20425,
            "unit": "allocs/op",
            "extra": "1008 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/large",
            "value": 35245075,
            "unit": "ns/op\t36549217 B/op\t  604598 allocs/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/large - ns/op",
            "value": 35245075,
            "unit": "ns/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/large - B/op",
            "value": 36549217,
            "unit": "B/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/large - allocs/op",
            "value": 604598,
            "unit": "allocs/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkValueTypeSwitchDictionary",
            "value": 743.3,
            "unit": "ns/op\t      64 B/op\t       1 allocs/op",
            "extra": "1619772 times\n4 procs"
          },
          {
            "name": "BenchmarkValueTypeSwitchDictionary - ns/op",
            "value": 743.3,
            "unit": "ns/op",
            "extra": "1619772 times\n4 procs"
          },
          {
            "name": "BenchmarkValueTypeSwitchDictionary - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "1619772 times\n4 procs"
          },
          {
            "name": "BenchmarkValueTypeSwitchDictionary - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "1619772 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_new_attribute",
            "value": 9.976,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120270655 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_new_attribute - ns/op",
            "value": 9.976,
            "unit": "ns/op",
            "extra": "120270655 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_new_attribute - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120270655 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_new_attribute - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120270655 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_an_existing_attribute",
            "value": 9.98,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120230799 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_an_existing_attribute - ns/op",
            "value": 9.98,
            "unit": "ns/op",
            "extra": "120230799 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_an_existing_attribute - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120230799 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_an_existing_attribute - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120230799 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_duplicate_attribute",
            "value": 9.968,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120415623 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_duplicate_attribute - ns/op",
            "value": 9.968,
            "unit": "ns/op",
            "extra": "120415623 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_duplicate_attribute - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120415623 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_duplicate_attribute - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120415623 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_hundred_locations_to_loop_through",
            "value": 13.4,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "88760232 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_hundred_locations_to_loop_through - ns/op",
            "value": 13.4,
            "unit": "ns/op",
            "extra": "88760232 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_hundred_locations_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "88760232 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_hundred_locations_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "88760232 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_new_location",
            "value": 9.971,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120319208 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_new_location - ns/op",
            "value": 9.971,
            "unit": "ns/op",
            "extra": "120319208 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_new_location - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120319208 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_new_location - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120319208 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesToProto",
            "value": 1843,
            "unit": "ns/op\t     312 B/op\t       2 allocs/op",
            "extra": "637296 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesToProto - ns/op",
            "value": 1843,
            "unit": "ns/op",
            "extra": "637296 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesToProto - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "637296 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesToProto - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "637296 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_hundred_stacks_to_loop_through",
            "value": 333.6,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "3582045 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_hundred_stacks_to_loop_through - ns/op",
            "value": 333.6,
            "unit": "ns/op",
            "extra": "3582045 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_hundred_stacks_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "3582045 times\n4 procs"
          },
          {
            "name": "BenchmarkSetStack/with_a_hundred_stacks_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "3582045 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=50%",
            "value": 130813063,
            "unit": "ns/op\t55986452 B/op\t  593505 allocs/op",
            "extra": "8 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=50% - ns/op",
            "value": 130813063,
            "unit": "ns/op",
            "extra": "8 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=50% - B/op",
            "value": 55986452,
            "unit": "B/op",
            "extra": "8 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMerge/overlap/overlap=50% - allocs/op",
            "value": 593505,
            "unit": "allocs/op",
            "extra": "8 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_duplicate_location",
            "value": 9.988,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120140496 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_duplicate_location - ns/op",
            "value": 9.988,
            "unit": "ns/op",
            "extra": "120140496 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_duplicate_location - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120140496 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_duplicate_location - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120140496 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_hundred_locations_to_loop_through",
            "value": 394,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "3057266 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_hundred_locations_to_loop_through - ns/op",
            "value": 394,
            "unit": "ns/op",
            "extra": "3057266 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_hundred_locations_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "3057266 times\n4 procs"
          },
          {
            "name": "BenchmarkSetLocation/with_a_hundred_locations_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "3057266 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesSwitchDictionary",
            "value": 1478,
            "unit": "ns/op\t     400 B/op\t       8 allocs/op",
            "extra": "807006 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesSwitchDictionary - ns/op",
            "value": 1478,
            "unit": "ns/op",
            "extra": "807006 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesSwitchDictionary - B/op",
            "value": 400,
            "unit": "B/op",
            "extra": "807006 times\n4 procs"
          },
          {
            "name": "BenchmarkProfilesSwitchDictionary - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "807006 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_an_existing_value",
            "value": 6.237,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "192481712 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_an_existing_value - ns/op",
            "value": 6.237,
            "unit": "ns/op",
            "extra": "192481712 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_an_existing_value - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "192481712 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_an_existing_value - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "192481712 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_new_function",
            "value": 4.991,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "240069716 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_new_function - ns/op",
            "value": 4.991,
            "unit": "ns/op",
            "extra": "240069716 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_new_function - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "240069716 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_new_function - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "240069716 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_an_existing_function",
            "value": 4.994,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "240255871 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_an_existing_function - ns/op",
            "value": 4.994,
            "unit": "ns/op",
            "extra": "240255871 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_an_existing_function - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "240255871 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_an_existing_function - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "240255871 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_duplicate_function",
            "value": 4.986,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "240657564 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_duplicate_function - ns/op",
            "value": 4.986,
            "unit": "ns/op",
            "extra": "240657564 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_duplicate_function - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "240657564 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_duplicate_function - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "240657564 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_hundred_functions_to_loop_through",
            "value": 6.076,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "197334309 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_hundred_functions_to_loop_through - ns/op",
            "value": 6.076,
            "unit": "ns/op",
            "extra": "197334309 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_hundred_functions_to_loop_through - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "197334309 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_hundred_functions_to_loop_through - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "197334309 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_duplicate_function",
            "value": 4.986,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "240657564 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_duplicate_function - ns/op",
            "value": 4.986,
            "unit": "ns/op",
            "extra": "240657564 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_duplicate_function - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "240657564 times\n4 procs"
          },
          {
            "name": "BenchmarkSetFunction/with_a_duplicate_function - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "240657564 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/with_refs",
            "value": 3225,
            "unit": "ns/op\t    1152 B/op\t       1 allocs/op",
            "extra": "369034 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/with_refs - ns/op",
            "value": 3225,
            "unit": "ns/op",
            "extra": "369034 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/with_refs - B/op",
            "value": 1152,
            "unit": "B/op",
            "extra": "369034 times\n4 procs"
          },
          {
            "name": "BenchmarkMarshalProfiles/small/with_refs - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "369034 times\n4 procs"
          },
          {
            "name": "BenchmarkScopeProfilesSwitchDictionary",
            "value": 20.86,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "57313771 times\n4 procs"
          },
          {
            "name": "BenchmarkScopeProfilesSwitchDictionary - ns/op",
            "value": 20.86,
            "unit": "ns/op",
            "extra": "57313771 times\n4 procs"
          },
          {
            "name": "BenchmarkScopeProfilesSwitchDictionary - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "57313771 times\n4 procs"
          },
          {
            "name": "BenchmarkScopeProfilesSwitchDictionary - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "57313771 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_duplicate_value",
            "value": 6.259,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "192504602 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_duplicate_value - ns/op",
            "value": 6.259,
            "unit": "ns/op",
            "extra": "192504602 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_duplicate_value - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "192504602 times\n4 procs"
          },
          {
            "name": "BenchmarkSetString/with_a_duplicate_value - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "192504602 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_new_attribute",
            "value": 9.976,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "120270655 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_new_attribute - ns/op",
            "value": 9.976,
            "unit": "ns/op",
            "extra": "120270655 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_new_attribute - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "120270655 times\n4 procs"
          },
          {
            "name": "BenchmarkSetAttribute/with_a_new_attribute - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "120270655 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=2000",
            "value": 5596756,
            "unit": "ns/op\t 2678586 B/op\t   36836 allocs/op",
            "extra": "214 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=2000 - ns/op",
            "value": 5596756,
            "unit": "ns/op",
            "extra": "214 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=2000 - B/op",
            "value": 2678586,
            "unit": "B/op",
            "extra": "214 times\n4 procs"
          },
          {
            "name": "BenchmarkDictionaryMergeHotKey/attrs=2000 - allocs/op",
            "value": 36836,
            "unit": "allocs/op",
            "extra": "214 times\n4 procs"
          },
          {
            "name": "BenchmarkLocationSwitchDictionary",
            "value": 1081,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkLocationSwitchDictionary - ns/op",
            "value": 1081,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkLocationSwitchDictionary - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkLocationSwitchDictionary - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/medium",
            "value": 1170744,
            "unit": "ns/op\t 1239317 B/op\t   20425 allocs/op",
            "extra": "1008 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/medium - ns/op",
            "value": 1170744,
            "unit": "ns/op",
            "extra": "1008 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/medium - B/op",
            "value": 1239317,
            "unit": "B/op",
            "extra": "1008 times\n4 procs"
          },
          {
            "name": "BenchmarkUnmarshalProfiles/medium - allocs/op",
            "value": 20425,
            "unit": "allocs/op",
            "extra": "1008 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashFourItems",
            "value": 206.8,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "5801455 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashFourItems - ns/op",
            "value": 206.8,
            "unit": "ns/op",
            "extra": "5801455 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashFourItems - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "5801455 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashFourItems - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "5801455 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashEightItems",
            "value": 516.7,
            "unit": "ns/op\t       8 B/op\t       1 allocs/op",
            "extra": "2320911 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashEightItems - ns/op",
            "value": 516.7,
            "unit": "ns/op",
            "extra": "2320911 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashEightItems - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "2320911 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashEightItems - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2320911 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashWithEmbeddedSliceAndMap",
            "value": 813.5,
            "unit": "ns/op\t       8 B/op\t       1 allocs/op",
            "extra": "1473560 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashWithEmbeddedSliceAndMap - ns/op",
            "value": 813.5,
            "unit": "ns/op",
            "extra": "1473560 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashWithEmbeddedSliceAndMap - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "1473560 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashWithEmbeddedSliceAndMap - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "1473560 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashWideMap",
            "value": 10062,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "118520 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashWideMap - ns/op",
            "value": 10062,
            "unit": "ns/op",
            "extra": "118520 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashWideMap - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "118520 times\n4 procs"
          },
          {
            "name": "BenchmarkMapHashWideMap - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "118520 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPRequest/HTTP/2.0,_client_per_thread_(like_single_app)",
            "value": 919397,
            "unit": "ns/op",
            "extra": "1174 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPRequest/HTTP/1.1,_client_per_thread_(like_single_app)",
            "value": 917873,
            "unit": "ns/op",
            "extra": "1156 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPRequest/HTTP/2.0,_shared_client_(like_load_balancer)",
            "value": 914902,
            "unit": "ns/op",
            "extra": "1136 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPRequest/HTTP/1.1,_shared_client_(like_load_balancer)",
            "value": 921876,
            "unit": "ns/op",
            "extra": "1166 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPRequest/HTTP/2.0,_client_per_thread_(like_single_app)",
            "value": 919397,
            "unit": "ns/op",
            "extra": "1174 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPRequest/HTTP/1.1,_client_per_thread_(like_single_app)",
            "value": 917873,
            "unit": "ns/op",
            "extra": "1156 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPRequest/HTTP/2.0,_shared_client_(like_load_balancer)",
            "value": 914902,
            "unit": "ns/op",
            "extra": "1136 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPRequest/HTTP/1.1,_shared_client_(like_load_balancer)",
            "value": 921876,
            "unit": "ns/op",
            "extra": "1166 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_metric_request/raw_bytes_183/compressed_bytes_138/compressor_zstd",
            "value": 7816,
            "unit": "ns/op",
            "extra": "152811 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_metric_request/raw_bytes_183/compressed_bytes_155/compressor_snappy",
            "value": 809.9,
            "unit": "ns/op",
            "extra": "1489533 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_metric_request/raw_bytes_376/compressed_bytes_193/compressor_gzip",
            "value": 26810,
            "unit": "ns/op",
            "extra": "44335 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_metric_request/raw_bytes_376/compressed_bytes_203/compressor_zstd",
            "value": 9048,
            "unit": "ns/op",
            "extra": "128346 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_log_request/raw_bytes_160/compressed_bytes_178/compressor_snappy",
            "value": 805.6,
            "unit": "ns/op",
            "extra": "1489764 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_log_request/raw_bytes_242/compressed_bytes_260/compressor_snappy",
            "value": 933.2,
            "unit": "ns/op",
            "extra": "1288774 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_log_request/raw_bytes_4850/compressed_bytes_463/compressor_snappy",
            "value": 1960,
            "unit": "ns/op",
            "extra": "608902 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_trace_request/raw_bytes_240/compressed_bytes_258/compressor_snappy",
            "value": 933.1,
            "unit": "ns/op",
            "extra": "1288123 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_metric_request/raw_bytes_10991/compressed_bytes_587/compressor_gzip",
            "value": 79462,
            "unit": "ns/op",
            "extra": "15061 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_metric_request/raw_bytes_10991/compressed_bytes_538/compressor_zstd",
            "value": 16917,
            "unit": "ns/op",
            "extra": "70480 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_log_request/raw_bytes_242/compressed_bytes_220/compressor_gzip",
            "value": 24330,
            "unit": "ns/op",
            "extra": "49122 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_log_request/raw_bytes_4850/compressed_bytes_217/compressor_zstd",
            "value": 12266,
            "unit": "ns/op",
            "extra": "96985 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_trace_request/raw_bytes_338/compressed_bytes_294/compressor_snappy",
            "value": 1017,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_trace_request/raw_bytes_7250/compressed_bytes_313/compressor_gzip",
            "value": 49032,
            "unit": "ns/op",
            "extra": "24226 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_metric_request/raw_bytes_376/compressed_bytes_215/compressor_snappy",
            "value": 971.1,
            "unit": "ns/op",
            "extra": "1233619 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_metric_request/raw_bytes_10991/compressed_bytes_1033/compressor_snappy",
            "value": 4232,
            "unit": "ns/op",
            "extra": "283088 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_log_request/raw_bytes_160/compressed_bytes_164/compressor_gzip",
            "value": 20760,
            "unit": "ns/op",
            "extra": "58660 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_log_request/raw_bytes_160/compressed_bytes_159/compressor_zstd",
            "value": 11119,
            "unit": "ns/op",
            "extra": "107024 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_log_request/raw_bytes_160/compressed_bytes_178/compressor_snappy",
            "value": 805.6,
            "unit": "ns/op",
            "extra": "1489764 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_log_request/raw_bytes_242/compressed_bytes_220/compressor_gzip",
            "value": 24330,
            "unit": "ns/op",
            "extra": "49122 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_log_request/raw_bytes_242/compressed_bytes_210/compressor_zstd",
            "value": 10996,
            "unit": "ns/op",
            "extra": "109034 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_log_request/raw_bytes_242/compressed_bytes_260/compressor_snappy",
            "value": 933.2,
            "unit": "ns/op",
            "extra": "1288774 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_log_request/raw_bytes_4850/compressed_bytes_253/compressor_gzip",
            "value": 35713,
            "unit": "ns/op",
            "extra": "33751 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_log_request/raw_bytes_4850/compressed_bytes_217/compressor_zstd",
            "value": 12266,
            "unit": "ns/op",
            "extra": "96985 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_log_request/raw_bytes_4850/compressed_bytes_463/compressor_snappy",
            "value": 1960,
            "unit": "ns/op",
            "extra": "608902 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_trace_request/raw_bytes_240/compressed_bytes_211/compressor_gzip",
            "value": 25914,
            "unit": "ns/op",
            "extra": "45538 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_trace_request/raw_bytes_240/compressed_bytes_211/compressor_zstd",
            "value": 8540,
            "unit": "ns/op",
            "extra": "137630 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_trace_request/raw_bytes_240/compressed_bytes_258/compressor_snappy",
            "value": 933.1,
            "unit": "ns/op",
            "extra": "1288123 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_trace_request/raw_bytes_338/compressed_bytes_256/compressor_gzip",
            "value": 27614,
            "unit": "ns/op",
            "extra": "43797 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_trace_request/raw_bytes_338/compressed_bytes_265/compressor_zstd",
            "value": 9362,
            "unit": "ns/op",
            "extra": "128020 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_trace_request/raw_bytes_338/compressed_bytes_294/compressor_snappy",
            "value": 1017,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_trace_request/raw_bytes_7250/compressed_bytes_313/compressor_gzip",
            "value": 49032,
            "unit": "ns/op",
            "extra": "24226 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_trace_request/raw_bytes_7250/compressed_bytes_269/compressor_zstd",
            "value": 11555,
            "unit": "ns/op",
            "extra": "103976 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_trace_request/raw_bytes_7250/compressed_bytes_615/compressor_snappy",
            "value": 2537,
            "unit": "ns/op",
            "extra": "454411 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_metric_request/raw_bytes_183/compressed_bytes_141/compressor_gzip",
            "value": 20474,
            "unit": "ns/op",
            "extra": "57453 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_metric_request/raw_bytes_183/compressed_bytes_138/compressor_zstd",
            "value": 7816,
            "unit": "ns/op",
            "extra": "152811 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_metric_request/raw_bytes_183/compressed_bytes_155/compressor_snappy",
            "value": 809.9,
            "unit": "ns/op",
            "extra": "1489533 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_metric_request/raw_bytes_376/compressed_bytes_193/compressor_gzip",
            "value": 26810,
            "unit": "ns/op",
            "extra": "44335 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_metric_request/raw_bytes_376/compressed_bytes_203/compressor_zstd",
            "value": 9048,
            "unit": "ns/op",
            "extra": "128346 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_metric_request/raw_bytes_376/compressed_bytes_215/compressor_snappy",
            "value": 971.1,
            "unit": "ns/op",
            "extra": "1233619 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_metric_request/raw_bytes_10991/compressed_bytes_587/compressor_gzip",
            "value": 79462,
            "unit": "ns/op",
            "extra": "15061 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_metric_request/raw_bytes_10991/compressed_bytes_538/compressor_zstd",
            "value": 16917,
            "unit": "ns/op",
            "extra": "70480 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_metric_request/raw_bytes_10991/compressed_bytes_1033/compressor_snappy",
            "value": 4232,
            "unit": "ns/op",
            "extra": "283088 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_log_request/raw_bytes_160/compressed_bytes_164/compressor_gzip",
            "value": 20760,
            "unit": "ns/op",
            "extra": "58660 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_log_request/raw_bytes_160/compressed_bytes_159/compressor_zstd",
            "value": 11119,
            "unit": "ns/op",
            "extra": "107024 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_log_request/raw_bytes_4850/compressed_bytes_253/compressor_gzip",
            "value": 35713,
            "unit": "ns/op",
            "extra": "33751 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_trace_request/raw_bytes_240/compressed_bytes_211/compressor_gzip",
            "value": 25914,
            "unit": "ns/op",
            "extra": "45538 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_trace_request/raw_bytes_240/compressed_bytes_211/compressor_zstd",
            "value": 8540,
            "unit": "ns/op",
            "extra": "137630 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_trace_request/raw_bytes_338/compressed_bytes_265/compressor_zstd",
            "value": 9362,
            "unit": "ns/op",
            "extra": "128020 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_trace_request/raw_bytes_7250/compressed_bytes_269/compressor_zstd",
            "value": 11555,
            "unit": "ns/op",
            "extra": "103976 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_log_request/raw_bytes_242/compressed_bytes_210/compressor_zstd",
            "value": 10996,
            "unit": "ns/op",
            "extra": "109034 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/md_trace_request/raw_bytes_338/compressed_bytes_256/compressor_gzip",
            "value": 27614,
            "unit": "ns/op",
            "extra": "43797 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/lg_trace_request/raw_bytes_7250/compressed_bytes_615/compressor_snappy",
            "value": 2537,
            "unit": "ns/op",
            "extra": "454411 times\n4 procs"
          },
          {
            "name": "BenchmarkCompressors/sm_metric_request/raw_bytes_183/compressed_bytes_141/compressor_gzip",
            "value": 20474,
            "unit": "ns/op",
            "extra": "57453 times\n4 procs"
          },
          {
            "name": "BenchmarkHTTPProtoLogsSequential",
            "value": 7200751,
            "unit": "ns/op",
            "extra": "159 times\n4 procs"
          },
          {
            "name": "BenchmarkGRPCLogsSequential",
            "value": 8020227,
            "unit": "ns/op",
            "extra": "162 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/no_parent",
            "value": 184.3,
            "unit": "ns/op\t     208 B/op\t       4 allocs/op",
            "extra": "6571111 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/no_parent - ns/op",
            "value": 184.3,
            "unit": "ns/op",
            "extra": "6571111 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/no_parent - B/op",
            "value": 208,
            "unit": "B/op",
            "extra": "6571111 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/no_parent - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "6571111 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/with_parent",
            "value": 284.8,
            "unit": "ns/op\t     352 B/op\t       7 allocs/op",
            "extra": "4233794 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/with_parent - ns/op",
            "value": 284.8,
            "unit": "ns/op",
            "extra": "4233794 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/with_parent - B/op",
            "value": 352,
            "unit": "B/op",
            "extra": "4233794 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/with_parent - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "4233794 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/no_parent",
            "value": 184.3,
            "unit": "ns/op\t     208 B/op\t       4 allocs/op",
            "extra": "6571111 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/no_parent - ns/op",
            "value": 184.3,
            "unit": "ns/op",
            "extra": "6571111 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/no_parent - B/op",
            "value": 208,
            "unit": "B/op",
            "extra": "6571111 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/no_parent - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "6571111 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/with_parent",
            "value": 284.8,
            "unit": "ns/op\t     352 B/op\t       7 allocs/op",
            "extra": "4233794 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/with_parent - ns/op",
            "value": 284.8,
            "unit": "ns/op",
            "extra": "4233794 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/with_parent - B/op",
            "value": 352,
            "unit": "B/op",
            "extra": "4233794 times\n4 procs"
          },
          {
            "name": "BenchmarkStartTracesOpLongLivedCtx/with_parent - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "4233794 times\n4 procs"
          },
          {
            "name": "BenchmarkBatchMetricProcessor2k",
            "value": 75045415,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkMultiBatchMetricProcessor2k",
            "value": 72999666,
            "unit": "ns/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkBatchMetricSplitMaxSize2k",
            "value": 30312061,
            "unit": "ns/op\t 5580162 B/op\t   49933 allocs/op",
            "extra": "46 times\n4 procs"
          },
          {
            "name": "BenchmarkBatchMetricSplitMaxSize2k - ns/op",
            "value": 30312061,
            "unit": "ns/op",
            "extra": "46 times\n4 procs"
          },
          {
            "name": "BenchmarkBatchMetricSplitMaxSize2k - B/op",
            "value": 5580162,
            "unit": "B/op",
            "extra": "46 times\n4 procs"
          },
          {
            "name": "BenchmarkBatchMetricSplitMaxSize2k - allocs/op",
            "value": 49933,
            "unit": "allocs/op",
            "extra": "46 times\n4 procs"
          },
          {
            "name": "BenchmarkTraceSizeBytes",
            "value": 406376,
            "unit": "ns/op",
            "extra": "2944 times\n4 procs"
          },
          {
            "name": "BenchmarkTraceSizeSpanCount",
            "value": 4.058,
            "unit": "ns/op",
            "extra": "295170122 times\n4 procs"
          }
        ]
      }
    ]
  }
}