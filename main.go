package main

import (
	commonCommunication "github.com/kulycloud/common/communication"
	"github.com/kulycloud/common/logging"
	"github.com/kulycloud/service-manager-k8s/communication"
	"github.com/kulycloud/service-manager-k8s/config"
	"github.com/kulycloud/service-manager-k8s/reconciling"
	"golang.org/x/net/context"
	"time"
)

var logger = logging.GetForComponent("init")

func main() {
	defer logging.Sync()

	err := config.ParseConfig()
	if err != nil {
		logger.Fatalw("Error parsing config", "error", err)
	}
	logger.Infow("Finished parsing config")

	ctx := context.Background()
	r, err := reconciling.NewReconciler()
	if err != nil {
		logger.Fatalw("could not connect to cluster: %w", err)
	}

	err = r.CheckAndSetup(ctx)
	if err != nil {
		logger.Fatalw("could not setup cluster: %w", err)
	}

	go registerLoop()

	logger.Info("Starting listener")
	listener := commonCommunication.NewListener(logging.GetForComponent("listener"))
	if err = listener.Setup(config.GlobalConfig.Port); err != nil {
		logger.Panicw("error initializing listener", "error", err)
	}

	handler := communication.NewServiceManagerHandler(r, listener)
	handler.Register()

	if err = listener.Serve(); err != nil {
		logger.Panicw("error serving listener", "error", err)
	}

	listener.Storage.Ready()
}

func registerLoop() {
	for {
		_, err := register()
		if err == nil {
			break
		}

		logger.Info("Retrying in 5s...")
		time.Sleep(5*time.Second)
	}
}

func register() (*commonCommunication.ControlPlaneCommunicator, error) {
	comm := commonCommunication.NewControlPlaneCommunicator()
	err := comm.Connect(config.GlobalConfig.ControlPlaneHost, config.GlobalConfig.ControlPlanePort)
	if err != nil {
		logger.Errorw("Could not connect to control-plane", "error", err)
		return nil, err
	}
	err = comm.RegisterThisService(context.Background(), "service-manager", config.GlobalConfig.Host, config.GlobalConfig.Port)
	if err != nil {
		logger.Errorw("Could not register service", "error", err)
		return nil, err
	}
	logger.Info("Registered to control-plane")
	return comm, nil
}
