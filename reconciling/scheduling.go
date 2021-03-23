package reconciling

import (
	"context"
	commonCommunication "github.com/kulycloud/common/communication"
)

const ResourceTypeService = "service"

type ReconcileScheduler struct {
	Reconciler Reconciler
	storage    *commonCommunication.StorageCommunicator
}

func NewReconcilerScheduler(storage *commonCommunication.StorageCommunicator, reconciler Reconciler) (*ReconcileScheduler, error) {
	return &ReconcileScheduler{
		Reconciler: reconciler,
		storage: storage,
	}, nil
}

func (scheduler *ReconcileScheduler) RegisterEventHandlers(controlPlane *commonCommunication.ControlPlaneCommunicator) error {
	err := controlPlane.RegisterStorageChangedHandler(scheduler.onStorageChangedEvent)
	if err != nil {
		return err
	}

	err = controlPlane.RegisterConfigurationChangedHandler(scheduler.onConfigurationChangedEvent)
	if err != nil {
		return err
	}

	return nil
}

func (scheduler *ReconcileScheduler) onConfigurationChangedEvent(event *commonCommunication.ConfigurationChanged) {
	if event.Resource.Type != ResourceTypeService {
		return
	}
	logger.Infow("reconciling namespace",
		"trigger", "event",
		"namespace", event.Resource.Namespace)

	err := scheduler.Reconciler.ReconcileDeployments(context.Background(), event.Resource.Namespace)
	if err != nil {
		logger.Errorw("error reconciling namespace",
			"trigger", "event",
			"namespace", event.Resource.Namespace,
			"error", err)
	}
}

func (scheduler *ReconcileScheduler) onStorageChangedEvent(event *commonCommunication.StorageChanged) {
	scheduler.Reconciler.PropagateStorageToLoadBalancers(context.Background(), event.Endpoints)
}

func (scheduler *ReconcileScheduler) Start() <-chan error {
	errStream := make(chan error)

	go func() {
		err := scheduler.Reconciler.MonitorCluster(context.Background())
		if err != nil {
			errStream <- err
		}
	}()

	// Periodically check for new namespaces and reconcile!

	return errStream
}
