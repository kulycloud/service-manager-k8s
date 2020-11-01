package communication

import (
	"context"
	"errors"
	commonCommunication "github.com/kulycloud/common/communication"
	"github.com/kulycloud/common/logging"
	protoCommon "github.com/kulycloud/protocol/common"
	protoServices "github.com/kulycloud/protocol/services"
	"github.com/kulycloud/service-manager-k8s/reconciling"
)

var _ protoServices.ServiceManagerServer = &ServiceManagerHandler{}

var ErrStorageNotReady = errors.New("storage is not ready")

var logger = logging.GetForComponent("handler")

type ServiceManagerHandler struct {
	protoServices.UnimplementedServiceManagerServer
	reconciler *reconciling.Reconciler
	listener *commonCommunication.Listener
}

func NewServiceManagerHandler(reconciler *reconciling.Reconciler, listener *commonCommunication.Listener) *ServiceManagerHandler {
	return &ServiceManagerHandler{
		reconciler: reconciler,
		listener: listener,
	}
}

func (handler *ServiceManagerHandler) Register() {
	protoServices.RegisterServiceManagerServer(handler.listener.Server, handler)
}

func (handler *ServiceManagerHandler) Reconcile(ctx context.Context, request *protoServices.ReconcileRequest) (*protoCommon.Empty, error) {
	if !handler.listener.Storage.Ready() {
		return nil, ErrStorageNotReady
	}

	logger.Info("Starting reconcile!")
	return &protoCommon.Empty{}, handler.reconciler.ReconcileDeployments(ctx, request.Namespace)
}
