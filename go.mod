module github.com/kulycloud/service-manager-k8s

go 1.15

require (
	github.com/kulycloud/common v1.0.0
	github.com/kulycloud/protocol v1.0.0
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
)

replace github.com/kulycloud/common v1.0.0 => ../common

replace github.com/kulycloud/protocol v1.0.0 => ../protocol
