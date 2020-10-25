package communication

import (
	"context"
	commonCommunication "github.com/kulycloud/common/communication"
	"github.com/kulycloud/common/logging"
	protoCommon "github.com/kulycloud/protocol/common"
	protoServices "github.com/kulycloud/protocol/services"
)

var _ protoServices.ServiceManagerServer = &ServiceManagerHandler{}

var logger = logging.GetForComponent("handler")

type ServiceManagerHandler struct {}

func NewServiceManagerHandler() *ServiceManagerHandler {
	return &ServiceManagerHandler{}
}

func (handler *ServiceManagerHandler) Register(listener *commonCommunication.Listener) {
	protoServices.RegisterServiceManagerServer(listener.Server, handler)
}

func (handler *ServiceManagerHandler) Reconcile(ctx context.Context, request *protoServices.ReconcileRequest) (*protoCommon.Empty, error) {
	go logger.Info("Starting reconcile!")
	return &protoCommon.Empty{}, nil
}
