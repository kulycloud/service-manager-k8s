package reconciling

import (
	"context"
	commonCommunication "github.com/kulycloud/common/communication"
	"time"
)

const ResourceTypeService = "service"
const ReconcilePeriod = 1 * time.Hour
const ReconcileCheckLoop = 5 * time.Minute
const ReconcileLoopErrorRetry = 1 * time.Minute
const TriggerPeriod = "period"
const TriggerEvent = "event"

type ReconcileScheduler struct {
	Reconciler      Reconciler
	storage         *commonCommunication.StorageCommunicator
	namespaces      map[string]time.Time
	stop            bool
	storageNotifier chan interface{}
}

func NewReconcilerScheduler(storage *commonCommunication.StorageCommunicator, reconciler Reconciler) (*ReconcileScheduler, error) {
	return &ReconcileScheduler{
		Reconciler: reconciler,
		storage:    storage,
		stop:       false,
		namespaces: make(map[string]time.Time),
		storageNotifier: make(chan interface{}),
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
	if !scheduler.storage.Ready() {
		return
	}

	scheduler.ReconcileNamespace(context.Background(), event.Resource.Namespace, TriggerEvent)
}

func (scheduler *ReconcileScheduler) onStorageChangedEvent(event *commonCommunication.StorageChanged) {
	if scheduler.storageNotifier != nil {
		go func() {
			select { // clear stream
				case <-scheduler.storageNotifier:
				default:
			}

			if scheduler.storage.Ready() {
				 scheduler.storageNotifier <- nil
			}
		}()
	}

	scheduler.Reconciler.PropagateStorageToLoadBalancers(context.Background(), event.Endpoints)
}

func (scheduler *ReconcileScheduler) ReconcileNamespace(ctx context.Context, namespace string, trigger string) {
	logger.Infow("reconciling namespace",
		"trigger", trigger,
		"namespace", namespace)
	err := scheduler.Reconciler.ReconcileDeployments(ctx, namespace)
	if err == nil {
		scheduler.namespaces[namespace] = time.Now()
	} else {
		logger.Errorw("error reconciling namespace",
			"trigger", "period",
			"namespace", namespace,
			"error", err)
	}
}

func (scheduler *ReconcileScheduler) needsReconcile(namespace string) bool {
	t, ok := scheduler.namespaces[namespace]
	return !ok || time.Now().Sub(t) >= ReconcilePeriod
}

func (scheduler *ReconcileScheduler) checkNamespaces(ctx context.Context) error {
	logger.Infow("reconciling namespaces",
		"trigger", TriggerPeriod)
	namespaces, err := scheduler.storage.GetNamespaces(ctx)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		if scheduler.needsReconcile(namespace) {
			scheduler.ReconcileNamespace(ctx, namespace, TriggerPeriod)
		}
	}
	return nil
}

func (scheduler *ReconcileScheduler) reconcileLoop() {
	ctx := context.Background()

	for !scheduler.stop {
		if !scheduler.storage.Ready() {
			logger.Warnw("trying to reconcile but storage is not ready")
			time.Sleep(ReconcileLoopErrorRetry)
			continue
		}
		err := scheduler.checkNamespaces(ctx)
		if err == nil {
			time.Sleep(ReconcileCheckLoop)
		} else {
			time.Sleep(ReconcileLoopErrorRetry)
		}
	}
}

func (scheduler *ReconcileScheduler) Start() <-chan error {
	errStream := make(chan error)

	go func() {
		// wait for storage to become available
		<- scheduler.storageNotifier
		scheduler.storageNotifier = nil

		for !scheduler.storage.Ready() {
			time.Sleep(10 * time.Second)
		}

		go func() {
			err := scheduler.Reconciler.MonitorCluster(context.Background())
			if err != nil {
				errStream <- err
			}
		}()

		scheduler.reconcileLoop()
	}()

	return errStream
}
