go_library(
    name = "summaryapireceiver",
    srcs = [
        "config.go",
        "doc.go",
        "factory.go",
        "metrics_receiver.go",
        "summary_api_collector.go",
    ],
)

go_test(
    name = "summaryapireceiver_test",
    size = "small",
    srcs = [
        "config_test.go",
        "factory_test.go",
    ],
    library = ":summaryapireceiver",
)
