load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")
load("@bazel_gazelle//:def.bzl", "gazelle")

gazelle(
    name = "gazelle",
    prefix = "github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server",
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server",
    visibility = ["//visibility:private"],
    deps = [
        "//gcloud:go_default_library",
        "//shell:go_default_library",
        "@com_github_gorilla_mux//:go_default_library",
        "@com_github_gorilla_websocket//:go_default_library",
    ],
)

go_binary(
    name = "server",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["main_test.go"],
    embed = [":go_default_library"],
)
