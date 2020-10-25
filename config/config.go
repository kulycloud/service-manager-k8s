package config

import (
	"errors"
	"fmt"
	commonConfig "github.com/kulycloud/common/config"
	"io/ioutil"
	"os"
)

type Config struct {
	Host string `configName:"host"`
	Port uint32 `configName:"port"`
	ControlPlaneHost string `configName:"controlPlaneHost"`
	ControlPlanePort uint32 `configName:"controlPlanePort"`
	KubeconfigContent string `configName:"kubeconfigContent" defaultValue:""`
	KubeconfigPath string `configName:"kubeconfigPath" defaultValue:""`
}

var GlobalConfig = &Config{}

var ErrNoKubeconfig = errors.New("no kubeconfig specified")

func ParseConfig() error {
	parser := commonConfig.NewParser()
	parser.AddProvider(commonConfig.NewCliParamProvider())
	parser.AddProvider(commonConfig.NewEnvironmentVariableProvider())

	err := parser.Populate(GlobalConfig)
	if err != nil {
		return err
	}

	if GlobalConfig.KubeconfigContent == "" {
		if GlobalConfig.KubeconfigPath == "" {
			return ErrNoKubeconfig
		}

		file, err := os.Open(GlobalConfig.KubeconfigPath)
		if err != nil {
			return fmt.Errorf("could not read kubeconfig: %w", err)
		}

		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			return fmt.Errorf("could not read kubeconfig: %w", err)
		}

		GlobalConfig.KubeconfigContent = string(bytes)

		err = file.Close()
		if err != nil {
			return fmt.Errorf("could not close kubeconfig: %w", err)
		}
	}

	return nil
}
