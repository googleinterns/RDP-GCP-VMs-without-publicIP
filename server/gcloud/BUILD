load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "executors.go",
        "runners.go",
        "types.go",
    ],
    importpath = "github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/gcloud",
    visibility = ["//visibility:public"],
    deps = [
        "//shell:go_default_library",
        "@com_github_gorilla_websocket//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "gcloud_test.go",
        "rdp_test.go",
    ],
    embed = [":go_default_library"],
    deps = ["@com_github_gorilla_websocket//:go_default_library"],
)
