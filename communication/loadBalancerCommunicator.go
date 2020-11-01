package communication

import (
	"context"
	"errors"
	"fmt"
	commonCommunication "github.com/kulycloud/common/communication"
	protoCommon "github.com/kulycloud/protocol/common"
	protoLoadBalancer "github.com/kulycloud/protocol/load-balancer"
	"strings"
)

type loadBalancerCommunicator struct {
	commonCommunication.ComponentCommunicator
	client protoLoadBalancer.LoadBalancerClient
}

type MultiLoadBalancerCommunicator []*loadBalancerCommunicator

var ErrMultiple = errors.New("multiple errors")

func NewLoadBalancerCommunicator(endpoint *protoCommon.Endpoint) (*loadBalancerCommunicator, error){
	comm, err := commonCommunication.NewComponentCommunicatorFromEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	cl := protoLoadBalancer.NewLoadBalancerClient(comm.GrpcClient)
	return &loadBalancerCommunicator{ComponentCommunicator: *comm, client: cl}, nil
}

func NewMultiLoadBalancerCommunicator(endpoints []*protoCommon.Endpoint) (MultiLoadBalancerCommunicator, error) {
	communicators := make([]*loadBalancerCommunicator, 0)
	errs := make([]error, 0)

	for _, endpoint := range endpoints {
		lbc, err := NewLoadBalancerCommunicator(endpoint)
		if err != nil {
			errs = append(errs, err)
		}
		communicators = append(communicators, lbc)
	}

	return communicators, mergeErrors(errs)
}

func (lbs MultiLoadBalancerCommunicator) RegisterStorageEndpoints(ctx context.Context, endpoints []*protoCommon.Endpoint) error {
	errs := make([]error, 0)

	for _, lbc := range lbs {
		err := lbc.RegisterStorageEndpoints(ctx, endpoints)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return mergeErrors(errs)
}

func (lbs MultiLoadBalancerCommunicator) SetEndpoints(ctx context.Context, endpoints []*protoCommon.Endpoint) error {
	el := &protoCommon.EndpointList{Endpoints: endpoints}
	errs := make([]error, 0)

	for _, lbc := range lbs {
		_, err := lbc.client.SetEndpoints(ctx, el)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return mergeErrors(errs)
}

func mergeErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	builder := strings.Builder{}
	for _, err := range errors {
		builder.WriteString(", ")
		builder.WriteString(err.Error())
	}

	return fmt.Errorf("%w: %s", ErrMultiple, builder.String())
}