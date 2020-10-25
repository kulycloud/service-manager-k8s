package config

import (
	commonConfig "github.com/kulycloud/common/config"
)

type Config struct {
	Host             string `configName:"host"`
	Port             uint32 `configName:"port"`
	ControlPlaneHost string `configName:"controlPlaneHost"`
	ControlPlanePort uint32 `configName:"controlPlanePort"`
	Kubeconfig       string `configName:"kubeconfig"`
	ServiceNamespace string `configName:"serviceNamespace" defaultValue:"kuly-services"`
}

var GlobalConfig = &Config{}

func ParseConfig() error {
	parser := commonConfig.NewParser()
	parser.AddProvider(commonConfig.NewCliParamProvider())
	parser.AddProvider(commonConfig.NewEnvironmentVariableProvider())

	return parser.Populate(GlobalConfig)
}
