load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "fake_nodeinfo.go",
        "kubelet_node_status_others.go",
        "kubelet_node_status_windows.go",
        "nodeinfo.go",
        "nodeinfo_reader.go",
        "nodeinfo_writter.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/nodeinfo",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/apis/core:go_default_library",
        "//pkg/apis/core/v1:go_default_library",
        "//pkg/util/node:go_default_library",
        "//pkg/util/taints:go_default_library",
        "//pkg/volume/util:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/clock:go_default_library",
        "//staging/src/k8s.io/client-go/kubernetes:go_default_library",
        "//staging/src/k8s.io/client-go/listers/core/v1:go_default_library",
        "//staging/src/k8s.io/cloud-provider:go_default_library",
        "//staging/src/k8s.io/cloud-provider/api:go_default_library",
        "//vendor/github.com/stretchr/testify/mock:go_default_library",
        "//vendor/k8s.io/klog/v2:go_default_library",
    ] + select({
        "@io_bazel_rules_go//go/platform:windows": [
            "//pkg/kubelet/winstats:go_default_library",
        ],
        "//conditions:default": [],
    }),
)

filegroup(
    name = "package-srcs",
    srcs = glob(["**"]),
    tags = ["automanaged"],
    visibility = ["//visibility:private"],
)

filegroup(
    name = "all-srcs",
    srcs = [":package-srcs"],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
