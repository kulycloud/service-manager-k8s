package reconciling

import (
	"context"
	"fmt"
	protoCommon "github.com/kulycloud/protocol/common"
	"github.com/kulycloud/service-manager-k8s/communication"
	"github.com/kulycloud/service-manager-k8s/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func (r *Reconciler) WatchPods(ctx context.Context) error {

	watchlist := cache.NewListWatchFromClient(
		r.clientset.CoreV1().RESTClient(),
		string(corev1.ResourcePods),
		config.GlobalConfig.ServiceNamespace,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		watchlist,
		&corev1.Pod{},
		0, //Duration is int64
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					logger.Warnw("could not cast")
					return
				}
				r.processPod(ctx, pod)
			},
			DeleteFunc: func(obj interface{}) {
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					logger.Warnw("could not cast")
					return
				}
				r.processPod(ctx, pod)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				pod, ok := newObj.(*corev1.Pod)
				if !ok {
					logger.Warnw("could not cast")
					return
				}
				r.processPod(ctx, pod)
			},
		},
	)

	stop := make(chan struct{})
	defer close(stop)
	controller.Run(stop)
	return nil
}

func (r *Reconciler) processPod(ctx context.Context, pod *corev1.Pod) {
	serviceName, ok := pod.Labels[nameLabel]
	if !ok {
		return // no serviceName set -> Not our pod
	}
	namespace, ok := pod.Labels[namespaceLabel]
	if !ok {
		return // no namespace set -> Not our pod
	}

	err := r.ReconcilePods(ctx, namespace, serviceName)
	if err != nil {
		logger.Warnw("error reconciling pods", "namespace", namespace, "serviceName", serviceName, "error", err, "triggeringPod", pod.Name)
	}
}

func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			if cond.Status != corev1.ConditionTrue {
				return false
			} else {
				return true
			}
		}
	}
	return false
}

func (r *Reconciler) getRunningPodEndpoints(ctx context.Context, namespace string, serviceName string, typeName string, port uint32) ([]*protoCommon.Endpoint, error) {
	pods, err := r.clientset.CoreV1().Pods(config.GlobalConfig.ServiceNamespace).List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s,%s=%s,%s=%s", namespaceLabel, namespace, nameLabel, serviceName, typeLabel, typeName)})
	if err != nil {
		return nil, err
	}

	ips := make([]*protoCommon.Endpoint, 0)

	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue // Pod being deleted
		}

		if !isPodReady(&pod) {
			continue // Pod not ready
		}

		if pod.Status.PodIP == "" {
			continue // Pod has no IP
		}

		ips = append(ips, &protoCommon.Endpoint{ Host: pod.Status.PodIP, Port: port})
	}

	return ips, nil
}

func (r *Reconciler) ReconcilePods(ctx context.Context, namespace string, serviceName string) error {
	lbs, err := r.getRunningPodEndpoints(ctx, namespace, serviceName, typeLabelLB, config.GlobalConfig.LoadBalancerControlPort)
	if err != nil {
		return err
	}

	err = r.storage.SetServiceLBEndpoints(ctx, namespace, serviceName, lbs)
	if err != nil {
		return fmt.Errorf("could not set LoadBalancers in storage: %w", err)
	}

	services, err := r.getRunningPodEndpoints(ctx, namespace, serviceName, typeLabelLB, config.GlobalConfig.HTTPPort)
	if err != nil {
		return err
	}

	communicator, err := communication.NewMultiLoadBalancerCommunicator(lbs)
	if err != nil {
		logger.Warnw("error connecting to load balancers", "error", err, "namespace", namespace, "service", serviceName)
	}

	err = communicator.SetEndpoints(ctx, services)

	if err != nil {
		logger.Warnw("error connecting to load balancers", "error", err, "namespace", namespace, "service", serviceName)
		return err
	}

	return nil
}
