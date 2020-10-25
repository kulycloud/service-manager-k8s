package reconciling

import (
	"context"
	"fmt"
	commonCommunication "github.com/kulycloud/common/communication"
	"github.com/kulycloud/common/logging"
	protoStorage "github.com/kulycloud/protocol/storage"
	"github.com/kulycloud/service-manager-k8s/config"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	labelPrefix      = "platform.kuly.cloud/"
	namespaceLabel   = labelPrefix + "namespace"
	typeLabel        = labelPrefix + "type"
	typeLabelService = "service"
	nameLabel 		 = labelPrefix + "name"
)

var logger = logging.GetForComponent("reconciler")

type Reconciler struct {
	clientset *kubernetes.Clientset
}

func NewReconciler() (*Reconciler, error) {
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
			Name: config.GlobalConfig.ServiceNamespace,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		return fmt.Errorf("could not create service namespace: %w", err)
	}
	return nil
}

func serviceDeploymentName(name *protoStorage.NamespacedName) string {
	return fmt.Sprintf("svc-%s-%s", name.Namespace, name.Name)
}

func pullSecretName(name *protoStorage.NamespacedName) string {
	return fmt.Sprintf("svc-%s-%s-pullsecret", name.Namespace, name.Name)
}

func buildPullSecrets(name *protoStorage.NamespacedName, service *protoStorage.Service) *corev1.Secret {
	data := []byte(service.PullSecrets)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta {
			Name: pullSecretName(name),
			Namespace: config.GlobalConfig.ServiceNamespace,
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": data,
		},
	}
}

func buildDeploymentFromService(name *protoStorage.NamespacedName, service *protoStorage.Service) *appsv1.Deployment {
	envVars := make([]corev1.EnvVar, 0)
	for name, value := range service.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name: name,
			Value: value,
		})
	}
	replicas := int32(service.Replicas)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceDeploymentName(name),
			Namespace: config.GlobalConfig.ServiceNamespace,
			Labels: map[string]string{
				namespaceLabel: name.Namespace,
				typeLabel:      typeLabelService,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					namespaceLabel: name.Namespace,
					typeLabel:      typeLabelService,
					nameLabel: 		name.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						namespaceLabel: name.Namespace,
						typeLabel:      typeLabelService,
						nameLabel: 		name.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "app-container",
							Image:   service.Image,
							Args:    service.Arguments,
							Ports: []corev1.ContainerPort{
								{
									Name: "grcp-port",
									ContainerPort: 30000,
								},
							},
							Env: envVars,
						},
					},
				},
			},
		},
	}

	if service.PullSecrets != "" {
		deployment.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", config.GlobalConfig.ServiceNamespace, pullSecretName(name)),
			},
		}
	}

	return &deployment
}

func (r *Reconciler) ReconcileNamespace(ctx context.Context, namespace string, storage *commonCommunication.StorageCommunicator) error {
	// Storage should be ready
	serviceNames, err := storage.GetServicesInNamespace(ctx, namespace)
	if err != nil {
		return err
	}

	deploymentsClient := r.clientset.AppsV1().Deployments(config.GlobalConfig.ServiceNamespace)
	secretsClient := r.clientset.CoreV1().Secrets(config.GlobalConfig.ServiceNamespace)

	deployments, err := deploymentsClient.List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s,%s=%s", typeLabel, typeLabelService, namespaceLabel, namespace)})
	if err != nil {
		return err
	}

	updated := make(map[string]bool)
	for _, dep := range deployments.Items {
		updated[dep.Name] = false
	}

	for _, name := range serviceNames {
		namespacedName := &protoStorage.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		serviceName := serviceDeploymentName(namespacedName)
		_, existing := updated[serviceName]
		updated[serviceName] = true

		service, err := storage.GetService(ctx, namespace, name)
		if err != nil {
			continue
		}

		if service.PullSecrets != "" {
			pullSecrets := buildPullSecrets(namespacedName, service)
			if existing {
				_, err = secretsClient.Update(ctx, pullSecrets, metav1.UpdateOptions{})
			} else {
				_, err = secretsClient.Create(ctx, pullSecrets, metav1.CreateOptions{})
			}
			if err != nil {
				logger.Warnw("Could not update/create PullSecret", "err", err, "existing", existing, "namespacedName", namespacedName)
				continue
			}
		}

		deployment := buildDeploymentFromService(namespacedName, service)
		if existing {
			_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
		} else {
			_, err = deploymentsClient.Create(ctx, deployment, metav1.CreateOptions{})
		}
		if err != nil {
			logger.Warnw("Could not update/create Deployment", "err", err, "existing", existing, "namespacedName", namespacedName)
			continue
		}
	}

	for name, handled := range updated {
		if handled == true {
			continue
		}

		// delete service that no longer exists
		namespacedName := &protoStorage.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := deploymentsClient.Delete(ctx, serviceDeploymentName(namespacedName), metav1.DeleteOptions{})
		if err != nil {
			logger.Warnw("Could not delete Deployment", "err", err, "namespacedName", namespacedName)
		}

		// This will probably throw errors quite often. Still cheaper than to check whether there are pull secrets before
		_ = secretsClient.Delete(ctx, pullSecretName(namespacedName), metav1.DeleteOptions{})
	}

	return nil
}
