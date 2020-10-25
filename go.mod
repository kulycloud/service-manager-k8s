module github.com/kulycloud/service-manager-k8s

go 1.15

require (
	github.com/kulycloud/common v1.0.0
	github.com/kulycloud/protocol v1.0.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
)

replace github.com/kulycloud/common v1.0.0 => ../common

replace github.com/kulycloud/protocol v1.0.0 => ../protocol
