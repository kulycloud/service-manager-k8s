package config

import (
	commonConfig "github.com/kulycloud/common/config"
)

type Config struct {
	Host                    string `configName:"host"`
	Port                    uint32 `configName:"port"`
	ControlPlaneHost        string `configName:"controlPlaneHost"`
	ControlPlanePort        uint32 `configName:"controlPlanePort"`
	Kubeconfig              string `configName:"kubeconfig" defaultValue:""`
	ServiceNamespace        string `configName:"serviceNamespace" defaultValue:"kuly-services"`
	LoadBalancerImage       string `configName:"loadBalancerImage" defaultValue:"kuly/loadbalancer"`
	LoadBalancerControlPort uint32 `configName:"loadBalancerControlPort" defaultValue:"12270"`
	HTTPPort                uint32 `configName:"httpPort" defaultValue:"30000"`
}

var GlobalConfig = &Config{}

func ParseConfig() error {
	parser := commonConfig.NewParser()
	parser.AddProvider(commonConfig.NewCliParamProvider())
	parser.AddProvider(commonConfig.NewEnvironmentVariableProvider())

	return parser.Populate(GlobalConfig)
}
