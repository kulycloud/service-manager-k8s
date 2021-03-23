package reconciling

import (
	"context"
	"fmt"
	commonCommunication "github.com/kulycloud/common/communication"
	"github.com/kulycloud/common/logging"
	protoCommon "github.com/kulycloud/protocol/common"
	"github.com/kulycloud/service-manager-k8s/config"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	labelPrefix      = "platform.kuly.cloud/"
	namespaceLabel   = labelPrefix + "namespace"
	typeLabel        = labelPrefix + "type"
	typeLabelService = "service"
	typeLabelLB      = "loadbalancer"
	nameLabel 		 = labelPrefix + "name"
)

var logger = logging.GetForComponent("reconciler")

type Reconciler interface {
	ReconcileDeployments(ctx context.Context, namespace string) error
	PropagateStorageToLoadBalancers(ctx context.Context, endpoints []*protoCommon.Endpoint)
	MonitorCluster(ctx context.Context) error
}

var _ Reconciler = &KubernetesReconciler{}

type KubernetesReconciler struct {
	storage *commonCommunication.StorageCommunicator
	clientset *kubernetes.Clientset
}

func NewKubernetesReconciler(storage *commonCommunication.StorageCommunicator) (*KubernetesReconciler, error) {
	var configObj *rest.Config
	var err error

	if config.GlobalConfig.Kubeconfig == "" {
		logger.Info("using in-cluster configuration for kubernetes")
		configObj, err = rest.InClusterConfig()
	} else {
		logger.Info("using provided kubeconfig")
		configObj, err = clientcmd.BuildConfigFromFlags("", config.GlobalConfig.Kubeconfig)
	}

	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(configObj)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &KubernetesReconciler{
		storage: storage,
		clientset: clientset,
	}, nil
}

func (r *KubernetesReconciler) CheckAndSetup(ctx context.Context) error {
	_, err := r.clientset.CoreV1().Namespaces().Get(ctx, config.GlobalConfig.ServiceNamespace, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return r.createNamespace(ctx)
		}
		return fmt.Errorf("get default namespace failed: %w", err)
	}
	return nil
}

func (r *KubernetesReconciler) createNamespace(ctx context.Context) error {
	logger.Info("creating namespace")
	_, err := r.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.GlobalConfig.ServiceNamespace,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		return fmt.Errorf("could not create service namespace: %w", err)
	}
	return nil
}