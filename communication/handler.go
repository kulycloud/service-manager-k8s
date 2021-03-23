package communication

import (
	commonCommunication "github.com/kulycloud/common/communication"
	protoServices "github.com/kulycloud/protocol/services"
)

var ControlPlane *commonCommunication.ControlPlaneCommunicator

var _ protoServices.ServiceManagerServer = &ServiceManagerHandler{}

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
