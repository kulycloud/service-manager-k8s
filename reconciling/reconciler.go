package reconciling

import (
	"context"
	"fmt"
	"github.com/kulycloud/common/logging"
	"github.com/kulycloud/service-manager-k8s/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var logger = logging.GetForComponent("reconciler")

type Reconciler struct {
	clientset *kubernetes.Clientset
}

func NewReconciler(ctx context.Context) (*Reconciler, error) {
	configObj, err := clientcmd.BuildConfigFromFlags("", config.GlobalConfig.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(configObj)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &Reconciler{
		clientset: clientset,
	}, nil
}

func (r *Reconciler) CheckAndSetup(ctx context.Context) error {
	_, err := r.clientset.CoreV1().Namespaces().Get(ctx, config.GlobalConfig.ServiceNamespace, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return r.createNamespace(ctx)
		}
		return fmt.Errorf("get default namespace failed: %w", err)
	}
	return nil
}

func (r *Reconciler) createNamespace(ctx context.Context) error {
	logger.Info("creating namespace")
	_, err := r.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:	   config.GlobalConfig.ServiceNamespace,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		return fmt.Errorf("could not create service namespace: %w", err)
	}
	return nil
}
