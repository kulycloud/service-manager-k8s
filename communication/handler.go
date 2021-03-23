package communication

import (
	"errors"
	commonCommunication "github.com/kulycloud/common/communication"
	"github.com/kulycloud/common/logging"
	protoServices "github.com/kulycloud/protocol/services"
)

var ControlPlane *commonCommunication.ControlPlaneCommunicator

var _ protoServices.ServiceManagerServer = &ServiceManagerHandler{}

var ErrStorageNotReady = errors.New("storage is not ready")

var logger = logging.GetForComponent("handler")

type ServiceManagerHandler struct {
	protoServices.UnimplementedServiceManagerServer
	listener *commonCommunication.Listener
}

func NewServiceManagerHandler(listener *commonCommunication.Listener) *ServiceManagerHandler {
	return &ServiceManagerHandler{
		listener: listener,
	}
}

func (handler *ServiceManagerHandler) Register() {
	protoServices.RegisterServiceManagerServer(handler.listener.Server, handler)
}
